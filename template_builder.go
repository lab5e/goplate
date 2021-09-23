package goplate

import (
	"errors"
	"fmt"
	"strings"
)

// TransformFunc is the transformation function for template expressions
type TransformFunc func(interface{}) []byte

// TransformFunctionMap is the function transform map for the templates.
type TransformFunctionMap map[string]TransformFunc

// Builder is an type to build and configure template instances.
type Builder struct {
	TemplateString string
	Transforms     TransformFunctionMap
	Parameters     interface{}
}

// New creates a new template builder
func New(templateString string) *Builder {
	ret := &Builder{
		TemplateString: templateString,
		Parameters:     nil,
		Transforms: TransformFunctionMap{
			"json":   DefaultJSONTransformFunc(DefaultMarshaler()),
			"asTime": Int64ToDateString,
			"hex":    HexConversion,
		},
	}

	return ret
}

// WithTransforms modifies the transform function map for the template. The
// transform function map is used to apply transformations to fields in the
// merged data structure. They are simple strings with no parameters and the
// key to the
func (t *Builder) WithTransforms(transformMap TransformFunctionMap) *Builder {
	for k, v := range transformMap {
		t.Transforms[k] = v
	}
	return t
}

// WithParameters sets the parameter type used for the template. When the structure
// is built it will use this parameter struct to validate the template
func (t *Builder) WithParameters(params interface{}) *Builder {
	t.Parameters = params
	return t
}

func (t *Builder) WithJSONMarshaler(marshaler JSONMarshaler) *Builder {
	t.Transforms["json"] = DefaultJSONTransformFunc(marshaler)
	return t
}

// Build builds and validates the template
func (t *Builder) Build() (*Template, error) {
	if t.TemplateString == "" || t.Parameters == nil {
		return nil, errors.New("missing parameters")
	}
	template, err := newTemplate(t.TemplateString, t.Transforms, t.Parameters)
	if err != nil {
		return nil, err
	}
	ok, errors := template.Validate()
	if !ok {
		return nil, fmt.Errorf(strings.Join(errors, ","))
	}
	return template, nil
}
