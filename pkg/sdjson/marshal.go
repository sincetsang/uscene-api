package sdjson

import (
	"encoding/json"

	"TMA/pkg/sderr"
)

// bytes

var (
	Unmarshal     = json.Unmarshal
	Marshal       = json.Marshal
	MarshalIndent = json.MarshalIndent
)

// string

func UnmarshalString(s string, v interface{}) error {
	err := json.Unmarshal([]byte(s), v)
	return sderr.WithStack(err)
}

func MarshalString(v interface{}) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", sderr.WithStack(err)
	}
	return string(raw), nil
}

func MarshalIndentString(v interface{}, prefix, indent string) (string, error) {
	raw, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		return "", sderr.WithStack(err)
	}
	return string(raw), nil
}

// value

func UnmarshalValue(raw []byte) (Value, error) {
	var v Value
	if err := json.Unmarshal(raw, &v); err != nil {
		return V(nil), sderr.WithStack(err)
	}
	return v, nil
}

func UnmarshalValueString(s string) (Value, error) {
	if v, err := UnmarshalValue([]byte(s)); err != nil {
		return V(nil), sderr.WithStack(err)
	} else {
		return v, nil
	}
}
