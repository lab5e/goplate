package template

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

type lookupInfo struct {
	FieldName    string
	FieldIndex   []int
	AccessorFunc stringAccessFunc
	NilValue     []byte
	IsMap        bool
	IsLeaf       bool
	Keep         bool // Keep field for later, set when field is checked
}

// The struct digger uses reflection to build a map with name -> function to
// retrieve values from the supplied data structures. The data structures
// doesn't have to be populated
type structDigger struct {
	funcs []*lookupInfo
}

func newStructDigger(templateData interface{}) *structDigger {
	ret := &structDigger{
		funcs: make([]*lookupInfo, 0),
	}
	ret.getFields(templateData, "")
	return ret
}

func (s *structDigger) RemoveUnusedFields() {
	var keepers []*lookupInfo
	for i, v := range s.funcs {
		if v.Keep {
			keepers = append(keepers, s.funcs[i])
		}
	}
	s.funcs = keepers
}

func (s *structDigger) KeepField(name string) {
	for i, v := range s.funcs {
		if v.FieldName == name {
			s.funcs[i].Keep = true
		}
	}
}

// HasField checks if the metadata contains the field
func (s *structDigger) HasField(name string) bool {
	for _, v := range s.funcs {
		if v.FieldName == name {
			return true
		}
	}
	return false
}

// GetField returns the field value
func (s *structDigger) GetField(name string, params interface{}) (interface{}, bool) {
	for _, info := range s.funcs {
		if info.FieldName == name {
			data := s.retrieveFieldValue(params, 0, info)
			return data, true
		}
	}
	return nil, false
}

// GetDataFunc returns a data func for the name. Unknown names will return
// a function that says "Unknown". Name must be lower case.
func (s *structDigger) GetValue(name string, params interface{}) ([]byte, bool) {
	for _, info := range s.funcs {
		if info.FieldName == name && info.IsLeaf {
			if info.IsMap {
				return []byte("[ value is a map ]"), false
			}
			data := s.retrieveFieldValue(params, 0, info)
			if data == nil {
				return info.NilValue, true
			}
			return []byte(info.AccessorFunc(data)), true
		}
	}
	return []byte("[ unknown field ]"), false
}

func (s *structDigger) GetMapValue(name string, key string, params interface{}) ([]byte, bool) {
	for _, info := range s.funcs {
		if info.FieldName == name {
			if !info.IsMap {
				return []byte("[ value not a map ] "), false
			}
			data, ok := s.retrieveFieldValue(params, 0, info).(map[string]string)
			if !ok {
				return []byte("[ invalid type ]"), false
			}
			ret, ok := data[key]
			return []byte(ret), ok
		}
	}
	return []byte("[ unknown map ]"), false
}

func (s *structDigger) appendField(name string, index []int, f stringAccessFunc, nilValue string, ismap bool) {
	info := &lookupInfo{
		FieldName:    strings.ToLower(name),
		FieldIndex:   make([]int, 0),
		AccessorFunc: f,
		NilValue:     []byte(nilValue),
		IsMap:        ismap,
		IsLeaf:       true,
		Keep:         false,
	}
	info.FieldIndex = append(info.FieldIndex, index...)
	s.funcs = append(s.funcs, info)
}

func (s *structDigger) appendParentField(name string, index []int) {
	info := &lookupInfo{
		FieldName:    strings.ToLower(name),
		FieldIndex:   make([]int, 0),
		AccessorFunc: nil,
		NilValue:     []byte{},
		IsMap:        false,
		IsLeaf:       false,
		Keep:         false,
	}
	info.FieldIndex = append(info.FieldIndex, index...)
	s.funcs = append(s.funcs, info)
}

func (s *structDigger) getFields(v interface{}, name string, fieldAccess ...int) {
	// Get the value that the value (might) point to
	val := reflect.Indirect(reflect.ValueOf(v))
	switch val.Kind() {
	case reflect.Struct:
		s.appendParentField(name, fieldAccess)
		// It's a struct so iterate across the fields in the struct.
		for fieldNum, field := range reflect.VisibleFields(val.Type()) {
			if !field.IsExported() {
				// Ignore unexported fields since they won't be visible to
				// the external users
				continue
			}
			// This is the fully qualified name of the field
			var fieldName string

			fieldName = fmt.Sprintf("%s.%s", name, field.Name)
			if name == "" {
				// Omit the period for the first name in the list
				fieldName = field.Name
			}

			var fieldVal reflect.Value
			//typeName := field.Type.String()
			if field.Type.Kind() == reflect.Ptr {
				// Create a new struct if this is a pointer
				fieldVal = reflect.New(field.Type.Elem())
			} else {
				// ..or just create the type
				fieldVal = reflect.New(field.Type)
			}
			// Append the field number to the array of indexes
			fields := append(fieldAccess, fieldNum)

			// This is just a simple optimization wrt the use of grpc-gateway and
			// the external interfaces. Normally this would appear like
			// device.deviceid.value in the list of fields but we cut this
			// short and just expose the "device.deviceid" field like in
			// grpc-gateway. Since these are used exclusively as pointers in
			// the API it's a shortcut.
			if field.Type.String() == "*wrapperspb.StringValue" {
				s.appendField(fieldName, fields, stringValueAccess, "", false)

				continue
			}
			if field.Type.String() == "*wrapperspb.Int32Value" {
				s.appendField(fieldName, fields, int32ValueAccess, "0", false)
				continue
			}
			if field.Type.String() == "*wrapperspb.Int64Value" {
				s.appendField(fieldName, fields, int64ValueAccess, "0", false)
				continue
			}
			if field.Type.String() == "*wrapperspb.BoolValue" {
				s.appendField(fieldName, fields, boolValueAccess, "false", false)
				continue
			}
			// ...and get the fields in this struct
			s.getFields(fieldVal.Elem().Interface(), fieldName, fields...)
		}

	case reflect.String:
		s.appendField(name, fieldAccess, stringAccess, "", false)

	case reflect.Map:
		s.appendField(name, fieldAccess, nil, "", true)

	case reflect.Bool:
		s.appendField(name, fieldAccess, boolAccess, "false", false)

	case reflect.Slice:
		switch reflect.TypeOf(v).Elem().Kind() {
		case reflect.Uint8:
			s.appendField(name, fieldAccess, byteSliceAccess, "", false)

		default:
			panic(fmt.Sprintf("Can't handle %s slices yet", reflect.TypeOf(v).Kind()))
		}
	case reflect.Int16:
		s.appendField(name, fieldAccess, int16Access, "0", false)

	case reflect.Int32:
		s.appendField(name, fieldAccess, int32Access, "0", false)

	case reflect.Int64:
		s.appendField(name, fieldAccess, int64Access, "0", false)

	case reflect.Int:
		s.appendField(name, fieldAccess, intAccess, "0", false)

	case reflect.Float32:
		s.appendField(name, fieldAccess, float32Access, "0.0", false)

	case reflect.Float64:
		s.appendField(name, fieldAccess, float64Access, "0.0", false)

	default:
		panic(fmt.Sprintf("Don't know how to handle types %s\n", val.Kind()))
	}
}

type stringAccessFunc func(interface{}) string

func stringAccess(val interface{}) string {
	return val.(string)
}

func boolAccess(val interface{}) string {
	if val.(bool) {
		return "true"
	}
	return "false"
}

func byteSliceAccess(val interface{}) string {
	buf := val.([]byte)
	return base64.StdEncoding.EncodeToString(buf)
}

func intAccess(val interface{}) string {
	return strconv.Itoa(int(val.(int)))
}

func int16Access(val interface{}) string {
	return strconv.Itoa(int(val.(int16)))
}

func int32Access(val interface{}) string {
	return strconv.Itoa(int(val.(int32)))
}

func int64Access(val interface{}) string {
	return strconv.FormatInt(val.(int64), 10)
}

func float32Access(val interface{}) string {
	return strconv.FormatFloat(float64(val.(float32)), 'f', 10, 32)
}

func float64Access(val interface{}) string {
	return strconv.FormatFloat(float64(val.(float64)), 'f', 10, 64)
}

func stringValueAccess(val interface{}) string {
	return val.(*wrapperspb.StringValue).Value
}

func int32ValueAccess(val interface{}) string {
	return strconv.Itoa(int(val.(*wrapperspb.Int32Value).Value))
}

func int64ValueAccess(val interface{}) string {
	return strconv.FormatInt(val.(*wrapperspb.Int64Value).Value, 10)
}

func boolValueAccess(val interface{}) string {
	if val.(*wrapperspb.BoolValue).Value {
		return "true"
	}
	return "false"
}

// This can probably be tuned a bit more if the output from the reflect package
// is cached. There's not a lot to gain here though since the reflect
// package is quite efficient.
func (s *structDigger) retrieveFieldValue(root interface{}, n int, data *lookupInfo) interface{} {

	fieldVal := reflect.ValueOf(root)
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			// Empty field, return a blank string (or nothing-string)
			return nil
		}
		fieldVal = fieldVal.Elem()
	}

	if len(data.FieldIndex) == n {
		// No more - use the field accessor
		// Pull the value here
		return root
	}
	return s.retrieveFieldValue(fieldVal.Field(data.FieldIndex[n]).Interface(), n+1, data)
}
