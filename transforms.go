package goplate

import (
	"encoding/hex"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

// DefaultJSONTransformFunc is the default JSON transform function
func DefaultJSONTransformFunc(marshaler JSONMarshaler) TransformFunc {
	return func(obj interface{}) []byte {
		buf, err := marshaler.Marshal(obj)
		if err != nil {
			return []byte("error")
		}
		return buf
	}
}

// Int64ToDateString returns the value as a RFC3339 date string. If the type isn't
// an int64 or *wrapperspb.Int64Value type it will return a blank
func Int64ToDateString(v interface{}) []byte {
	if v == nil {
		return []byte{}
	}
	val, ok := v.(int64)
	if !ok {
		wp, ok := v.(*wrapperspb.Int64Value)
		if !ok {
			return []byte{}
		}
		ts := time.Unix(0, wp.Value)
		return []byte(ts.Format(time.RFC3339))
	}
	return []byte(time.Unix(0, val).Format(time.RFC3339))
}

// HexConversion returns a hex-encoded string
func HexConversion(v interface{}) []byte {
	buf, ok := v.([]byte)
	if !ok {
		return []byte{}
	}
	return []byte(hex.EncodeToString(buf))
}
