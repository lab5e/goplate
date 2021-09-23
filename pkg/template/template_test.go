package template

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"testing"
	gotemplate "text/template"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestTemplateRendering(t *testing.T) {
	assert := require.New(t)

	tmpl, err := New(`{{ substructure.string }}/{{ string }}/{{ substructure.map["nAmE"] }}`).WithParameters(&testStructure{}).Build()
	assert.NoError(err)
	assert.NotNil(tmpl)

	params := testStructure{
		Substructure: &testSubStructure{
			String: &wrapperspb.StringValue{Value: "stringValue"},
			Map: map[string]string{
				"name": "the item name",
			},
		},
	}
	buf := &bytes.Buffer{}
	assert.NoError(tmpl.Execute(buf, &params))
	assert.Equal(fmt.Sprintf("%s/%s/%s",
		params.Substructure.String.Value,
		params.String,
		params.Substructure.Map["name"]),
		buf.String())

	tmpl, err = New(`:{{substructure.map["notag"]}}:{{substructure.map["notag"]}}:`).WithParameters(&testStructure{}).Build()
	assert.NoError(err)
	assert.NotNil(tmpl)
	buf = &bytes.Buffer{}
	assert.NoError(tmpl.Execute(buf, &params))
	assert.Equal(":::", buf.String())
}

func TestTransformCheck(t *testing.T) {
	assert := require.New(t)

	found, elem, transforms := hasTransforms("something")
	assert.False(found)
	assert.Equal("something", elem)
	assert.Equal("", transforms)

	found, elem, transforms = hasTransforms("something | other")
	assert.True(found)
	assert.Equal("something", elem)
	assert.Equal("other", transforms)

}

// Test payload rendering. This is a bit different from the string-type replacements for the topics
// elsewhere. The objects uses the default marshaller.
func TestTransformFunctions(t *testing.T) {
	assert := require.New(t)

	marshaler := DefaultMarshaler()

	tmpl, err := New(`{{ substructure | json }}`).WithParameters(&testStructure{}).Build()
	assert.NoError(err)

	params := &testStructure{
		Substructure: &testSubStructure{
			Bool:   true,
			Binary: []byte("some binary"),
			Map: map[string]string{
				"name": "value",
			},
		},
	}

	buf := &bytes.Buffer{}
	assert.NoError(tmpl.Execute(buf, params))
	expected, err := marshaler.Marshal(params.Substructure)
	assert.NoError(err)

	assert.Equal(expected, buf.Bytes())

	tmpl, err = New("{{int64 | asTime }}").WithParameters(&testStructure{}).Build()
	assert.NoError(err)

	ts := time.Now()
	testStruct := &testStructure{
		Int64:        ts.UnixNano(),
		Int64Wrapper: &wrapperspb.Int64Value{Value: ts.UnixNano()},
	}

	buf.Reset()
	assert.NoError(tmpl.Execute(buf, testStruct))
	assert.Equal(ts.Format(time.RFC3339), buf.String())

	tmpl, err = New("{{int64wrapper | asTime }}").WithParameters(&testStructure{}).Build()
	assert.NoError(err)
	buf.Reset()
	assert.NoError(tmpl.Execute(buf, testStruct))
	assert.Equal(ts.Format(time.RFC3339), buf.String())

	params.Substructure.Binary = []byte{0xbe, 0xef, 0xba, 0xbe}
	tmpl, err = New("{{substructure.binary | hex }}").WithParameters(&testStructure{}).Build()
	assert.NoError(err)
	buf.Reset()
	assert.NoError(tmpl.Execute(buf, params))

	assert.Equal("beefbabe", buf.String())
}

func TestCustomTransforms(t *testing.T) {
	assert := require.New(t)

	marshaler := DefaultMarshaler()

	tmpl, err := New("{{ substructure.subsub | incr }}").
		WithParameters(&testStructure{}).
		WithTransforms(TransformFunctionMap{
			"incr": func(v interface{}) []byte {
				val, ok := v.(*testSubSubStructure)
				if !ok {
					return []byte{}
				}
				val.Int++
				buf, err := marshaler.Marshal(val)
				if err != nil {
					return []byte{}
				}
				return buf
			},
		}).Build()
	assert.NoError(err)

	params := &testStructure{
		Substructure: &testSubStructure{
			SubSub: &testSubSubStructure{Int: 1},
		},
	}
	buf := &bytes.Buffer{}
	assert.NoError(tmpl.Execute(buf, params))

	// This field has been modified by the incr transform above
	params.Substructure.SubSub.Int = 2
	expected, err := marshaler.Marshal(params.Substructure.SubSub)
	assert.NoError(err)

	assert.Equal(expected, buf.Bytes())
}

func TestMapCheck(t *testing.T) {
	assert := require.New(t)

	found, name, key := isMapLookup(`some.map["name"]`)
	assert.True(found)
	assert.Equal("some.map", name)
	assert.Equal("name", key)
}

func TestTemplateValidation(t *testing.T) {
	assert := require.New(t)
	tmpl, err := newTemplate(`{{int32}} {{substructure.float64}} {{mumbojump{}foo}}`, make(TransformFunctionMap), &testStructure{})
	assert.NoError(err)
	assert.NotNil(tmpl)

	// Should return two errors
	ok, errors := tmpl.Validate()
	assert.False(ok)
	assert.Len(errors, 2)

	tmpl, err = newTemplate(`{{int32}} {{substructure.bool}} {{substructure.map["name"]}}`, make(TransformFunctionMap), &testStructure{})
	assert.NoError(err)
	assert.NotNil(tmpl)
	ok, errors = tmpl.Validate()
	assert.True(ok)
	assert.Len(errors, 0)
}

func TestTemplateSyntaxValidation(t *testing.T) {
	assert := require.New(t)
	_, err := New(`{{int64}} {{int32}} {{bool`).WithParameters(&testStructure{}).Build()
	assert.Error(err)
}

func BenchmarkGoTemplate(b *testing.B) {
	tmpl := gotemplate.Must(gotemplate.New("test").Parse(`
	{
		"field1": "{{ .Int32 }}",
		"field2": "{{ .Substructure.SubSub.Int }}",
		"field3": "{{ .Substructure.Bool }}",
		"field4": "{{ index .Substructure.Map "name" }}"
	}
	`))

	params := &testStructure{
		Int32: 1, Int64: 2,
		Substructure: &testSubStructure{
			Bool: true,
			Map: map[string]string{
				"name": "value",
			},
			SubSub: &testSubSubStructure{
				Int: 3,
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tmpl.Execute(io.Discard, params); err != nil {
			b.Fatalf("Error %v", err)
		}
	}
}

func BenchmarkCustomTemplate(b *testing.B) {
	tmpl, err := New(`
	{
		"field1": "{{ int32 }}",
		"field2": "{{ substructure.subSub.Int }}",
		"field3": "{{ Substructure.Bool }}",
		"field4": "{{ Substructure.Map["name"] }}"
	}
	`).WithParameters(&testStructure{}).Build()
	if err != nil {
		b.FailNow()
	}
	params := &testStructure{
		Int32: 1, Int64: 2,
		Substructure: &testSubStructure{
			Bool: true,
			Map: map[string]string{
				"name": "value",
			},
			SubSub: &testSubSubStructure{
				Int: 3,
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tmpl.Execute(io.Discard, params); err != nil {
			b.Fatalf("Error: %v", err)
		}
	}
}

func BenchmarkCustomTemplateTopicStatic(b *testing.B) {
	tmpl, err := New(`template/with/static/content`).WithParameters(&testStructure{}).Build()
	if err != nil {
		b.FailNow()
	}
	params := &testStructure{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Execute(io.Discard, params)
	}
}
