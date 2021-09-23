# goplate - yet another go template library

Fields are enclosed in double curly braces like regular Go templates:

    {{ fieldName }}

Nested data structures use dot separators.

    {{ fieldName.subStructure.field }}

Template syntax is fairly obvious - the names match roughly the JSON
equivalent when marshalled. Names are case insensitive so `{{ fieldName }}` and
`{{ fieldname }}` resolves to the same field.

Map access is limited to `map[string]string` only and map access is also
fairly obvious if you are familiar with Go:

    {{ mapName["name"] }}

## Transformation functions


There is a few transformations available in the templates. This marshals the
entire field as a JSON structure and includes it in the output

    {{ fieldName | json }}

This formats an int64 nanosecond field to a string representation of the data:

    {{ timestampField | asTime }}


This formats a byte buffer to a hex string:

    {{ byteField | asHex }}

## Usage

This will build a template and execute it:

    type testStruct struct {
        Field1 int
        Field2 int
    }

    // ...

    tmpl, err := New(`{{ field1 }}/{{ field2 }}`).
        WithParameters(&testStruct{}).
        Build()
    if err != nil {
        panic()
    }

    buf := &bytes.Buffer{}
    if err := tmpl.Execute(buf, &testStruct{Field1: 1, Field2: 2}) {
        panic()
    }

    // Print buffer
    fmt.Println(buf.String())
