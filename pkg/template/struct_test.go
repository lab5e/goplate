package template

import "google.golang.org/protobuf/types/known/wrapperspb"

// Test data structures

type testSubSubStructure struct {
	Int     int
	Float32 float32
	Float64 float64
}
type testSubStructure struct {
	Bool           bool
	SubSub         *testSubSubStructure
	String         *wrapperspb.StringValue
	Binary         []byte
	Map            map[string]string
	UnsupportedMap map[int32]string
}

type testStructure struct {
	Int16        int16
	Int32        int32
	Int64        int64
	Int32Wrapper *wrapperspb.Int32Value
	Int64Wrapper *wrapperspb.Int64Value
	BoolWrapper  *wrapperspb.BoolValue
	String       string
	Substructure *testSubStructure
}
