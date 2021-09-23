package goplate

import "encoding/json"

func DefaultMarshaler() JSONMarshaler {
	return &defaultJSONMarshaler{}
}

type JSONMarshaler interface {
	Marshal(obj interface{}) ([]byte, error)
}

type defaultJSONMarshaler struct {
}

func (d *defaultJSONMarshaler) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}
