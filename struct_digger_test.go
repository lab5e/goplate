package goplate

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestStructDigger(t *testing.T) {
	assert := require.New(t)

	sd := newStructDigger(&testStructure{})
	assert.NotNil(sd)

	testData := &testStructure{
		Int16:         16,
		Int32:         32,
		Int64:         64,
		Int32Wrapper:  &wrapperspb.Int32Value{Value: 132},
		Int64Wrapper:  &wrapperspb.Int64Value{Value: 164},
		BoolWrapper:   &wrapperspb.BoolValue{Value: true},
		ArrayOfint32:  []int32{1, 2, 3, 4},
		ArrayOfuint32: []uint32{1, 2, 3, 4},
		ArrayOfint64:  []int64{1, 2, 3, 4},
		ArrayOfuint64: []uint64{1, 2, 3, 4},
		Substructure: &testSubStructure{
			Bool:   true,
			String: &wrapperspb.StringValue{Value: "str"},
			Binary: []byte("binary"),
			Map: map[string]string{
				"name": "somename",
			},
			SubSub: &testSubSubStructure{
				Int:     3,
				Float32: 32.0,
				Float64: 64.0,
			},
		},
	}

	buf, found := sd.GetValue("int16", testData)
	assert.True(found)
	assert.Equal([]byte("16"), buf)

	buf, found = sd.GetValue("int32", testData)
	assert.True(found)
	assert.Equal([]byte("32"), buf)

	_, found = sd.GetValue("unknown", testData)
	assert.False(found)

	buf, found = sd.GetValue("int64", testData)
	assert.True(found)
	assert.Equal([]byte("64"), buf)

	buf, found = sd.GetValue("int32wrapper", testData)
	assert.True(found)
	assert.Equal([]byte("132"), buf)

	buf, found = sd.GetValue("int64wrapper", testData)
	assert.True(found)
	assert.Equal([]byte("164"), buf)

	buf, found = sd.GetValue("substructure.subsub.int", testData)
	assert.True(found)
	assert.Equal([]byte("3"), buf)

	buf, found = sd.GetValue("substructure.subsub.float32", testData)
	assert.True(found)
	assert.Equal([]byte("32.0000000000"), buf)

	// Map elements can't be accessed as values
	_, found = sd.GetValue("device.tags", testData)
	assert.False(found)

	// partials are not found
	_, found = sd.GetValue("substructure.subsub", testData)
	assert.False(found)

	buf, found = sd.GetValue("substructure.subsub.float64", testData)
	assert.True(found)
	assert.Equal([]byte("64.0000000000"), buf)

	buf, found = sd.GetValue("substructure.bool", testData)
	assert.True(found)
	assert.Equal([]byte("true"), buf)

	buf, found = sd.GetValue("boolwrapper", testData)
	assert.True(found)
	assert.Equal([]byte("true"), buf)

	testData.Substructure.String = &wrapperspb.StringValue{Value: "str"}
	buf, found = sd.GetValue("substructure.string", testData)
	assert.True(found)
	assert.Equal([]byte("str"), buf)

	testData.Substructure.Map = make(map[string]string)
	testData.Substructure.Map["name"] = "somename"

	// tag lookups work
	buf, found = sd.GetMapValue("substructure.map", "name", testData)
	assert.True(found)
	assert.Equal([]byte("somename"), buf)

	// non-map types return not found
	_, found = sd.GetMapValue("substructure.bool", "foo", testData)
	assert.False(found)

	// Unsupported map types returns not found
	_, found = sd.GetMapValue("unsupportedmap", "name", testData)
	assert.False(found)

	// Unknown fields returns not found
	_, found = sd.GetMapValue("message.device.metadata.unknown", "name", testData)
	assert.False(found)

	// Slices work just fine
	const payload = "some payload string"
	testData.Substructure.Binary = []byte(payload)
	expected := base64.StdEncoding.EncodeToString(testData.Substructure.Binary)
	buf, found = sd.GetValue("substructure.binary", testData)
	assert.True(found)
	assert.Equal(expected, string(buf))

	testData.ArrayOfint32 = []int32{1, 2, 3}
	expected = "[1,2,3]"
	buf, found = sd.GetValue("arrayofint32", testData)
	assert.True(found)
	assert.Equal(expected, string(buf))

	testData.ArrayOfuint32 = []uint32{1, 2, 3}
	expected = "[1,2,3]"
	buf, found = sd.GetValue("arrayofuint32", testData)
	assert.True(found)
	assert.Equal(expected, string(buf))

	testData.ArrayOfint64 = []int64{1, 2, 3}
	expected = "[1,2,3]"
	buf, found = sd.GetValue("arrayofint64", testData)
	assert.True(found)
	assert.Equal(expected, string(buf))

	testData.ArrayOfuint64 = []uint64{1, 2, 3}
	expected = "[1,2,3]"
	buf, found = sd.GetValue("arrayofuint64", testData)
	assert.True(found)
	assert.Equal(expected, string(buf))
}

func BenchmarkFieldRetrieve(b *testing.B) {
	d := newStructDigger(&testStructure{})

	testData := &testStructure{
		Substructure: &testSubStructure{
			SubSub: &testSubSubStructure{
				Int: 3,
			},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, found := d.GetValue("substructure.subsub.int", testData)
		if !found {
			b.Fatal("not found")
		}
		if string(v) != "3" {
			b.Fatal("v = ", v)
		}
	}
}
