package sdjson

import (
	"encoding/json"

	"TMA/pkg/sderr"
)

//

func MustUnmarshal(raw []byte, v interface{}) {
	if err := json.Unmarshal(raw, v); err != nil {
		panic(sderr.WithStack(err))
	}
}

func MustMarshal(v interface{}) []byte {
	if raw, err := json.Marshal(v); err != nil {
		panic(sderr.WithStack(err))
	} else {
		return raw
	}
}

func MustMarshalIndent(v interface{}, prefix, indent string) []byte {
	if raw, err := json.MarshalIndent(v, prefix, indent); err != nil {
		panic(sderr.WithStack(err))
	} else {
		return raw
	}
}

// string

func MustUnmarshalString(s string, v interface{}) {
	MustUnmarshal([]byte(s), v)
}

func MustMarshalString(v interface{}) string {
	return string(MustMarshal(v))
}

func MustMarshalIndentString(v interface{}, prefix, indent string) string {
	return string(MustMarshalIndent(v, prefix, indent))
}

// value

func MustUnmarshalValue(raw []byte) Value {
	if v, err := UnmarshalValue(raw); err != nil {
		panic(sderr.WithStack(err))
	} else {
		return v
	}
}

func MustUnmarshalValueString(s string) Value {
	if v, err := UnmarshalValueString(s); err != nil {
		panic(sderr.WithStack(err))
	} else {
		return v
	}
}
