package template

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// The entire template is build from a list of functions called in
// sequence to assemble the template.
type sectionFunc func(writer io.Writer, params interface{})

// Template is the main templating engine
type Template struct {
	metadata           *structDigger
	renderingFunctions []sectionFunc
	expressions        []string
	transformFunctions TransformFunctionMap
}

type state int

const (
	insideTag state = iota
	outsideTag
)

func staticElementFunc(field string) sectionFunc {
	b := []byte(field)
	return func(writer io.Writer, params interface{}) {
		if len(field) > 0 {
			_, _ = writer.Write(b)
		}
	}
}

var tagMatch *regexp.Regexp

func init() {
	tagMatch = regexp.MustCompile(`(.*)\[\"(.*)\"\]`)
}

// Checks if this is a map lookup and returns true/false and the tag + key names.
func isMapLookup(field string) (bool, string, string) {
	elems := tagMatch.FindAllStringSubmatch(field, -1)
	if len(elems) != 1 {
		return false, "", ""
	}
	return true, elems[0][1], elems[0][2]
}

// Checks if this field has a transform pipe set and returns the tag + names of
// transforms. The returned names are trimmed for surrounding whitespace.
func hasTransforms(field string) (bool, string, string) {
	elems := strings.Split(field, "|")
	if len(elems) < 2 {
		return false, field, ""
	}
	for i, v := range elems {
		elems[i] = strings.TrimSpace(v)
	}
	return true, elems[0], strings.Join(elems[1:], "|")
}

// tagElementFunc returns a function that will return the contents of
// the element tag.
func tagElementFunc(tag string, digger *structDigger, transformFunctions TransformFunctionMap) sectionFunc {
	tagLC := strings.ToLower(tag)
	// Check if this is a map lookup
	istag, name, key := isMapLookup(tagLC)
	if istag {
		digger.KeepField(name)
		return func(writer io.Writer, params interface{}) {
			buf, found := digger.GetMapValue(name, key, params)
			if !found {
				// Write nothing
				return
			}
			_, _ = writer.Write(buf)
		}
	}

	isTransform, name, funcs := hasTransforms(tag)
	if isTransform {
		// Find matching transform
		transformFunc, ok := transformFunctions[funcs]
		if !ok {
			// Return null function
			return func(io.Writer, interface{}) {}
		}
		digger.KeepField(name)
		return func(writer io.Writer, params interface{}) {
			val, found := digger.GetField(name, params)
			if !found || val == nil {
				return
			}
			_, _ = writer.Write(transformFunc(val))
		}
	}

	// A regular leaf node field that gets merged
	digger.KeepField(tagLC)
	return func(writer io.Writer, params interface{}) {
		buf, found := digger.GetValue(tagLC, params)
		if !found {
			// Write nothing
			return
		}
		_, _ = writer.Write(buf)
	}
}

// newTemplate creates a new templat
func newTemplate(templateStr string, transforms TransformFunctionMap, params interface{}) (*Template, error) {
	funcs := make([]sectionFunc, 0)

	metadata := newStructDigger(params)
	start := 0
	state := outsideTag
	prevCh := ' '
	expressions := make([]string, 0)
	// Scan through the template string and find strings that should be replaced
	for i, ch := range templateStr {
		if ch == '{' && prevCh == '{' {
			// Add preceeding bytes to function list
			if i > 0 {
				funcs = append(funcs, staticElementFunc(templateStr[start:i-1]))
			}
			start = i + 1
			state = insideTag
		}
		if ch == '}' && prevCh == '}' {
			// End of the tag - add contents of tag to function list
			//
			tag := strings.TrimSpace(templateStr[start : i-1])
			funcs = append(funcs, tagElementFunc(tag, metadata, transforms))
			state = outsideTag
			start = i + 1
			expressions = append(expressions, strings.ToLower(tag))
		}
		prevCh = ch
	}
	funcs = append(funcs, staticElementFunc(templateStr[start:]))
	// Add remainder of template to function list
	if state == insideTag {
		return nil, fmt.Errorf("template parse error (tag at %d isn't closed)", start)
	}
	metadata.RemoveUnusedFields()
	return &Template{
		renderingFunctions: funcs,
		metadata:           metadata,
		expressions:        expressions,
		transformFunctions: transforms,
	}, nil
}

// Execute writes the expanded template to the supplied io.Writer
func (t *Template) Execute(writer io.Writer, params interface{}) error {
	for _, f := range t.renderingFunctions {
		f(writer, params)
	}
	return nil
}

// Validate validates the template tags
func (t *Template) Validate() (bool, []string) {
	errs := make([]string, 0)
	for _, v := range t.expressions {
		expr := strings.ToLower(v)
		ismap, tag, _ := isMapLookup(expr)
		if ismap {
			expr = tag
		}
		istransform, tag, _ := hasTransforms(expr)
		if istransform {
			expr = tag
		}
		// TODO: Check transform functions
		if exists := t.metadata.HasField(expr); !exists {
			errs = append(errs, expr+" is not a known expression")
		}
	}
	return len(errs) == 0, errs
}
