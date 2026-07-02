package sdjson

func MarshalStringDef(v interface{}, def string) string {
	if r, err := MarshalString(v); err != nil {
		return def
	} else {
		return r
	}
}

func MarshalIndentStringDef(v interface{}, prefix, indent, def string) string {
	if r, err := MarshalIndentString(v, prefix, indent); err != nil {
		return def
	} else {
		return r
	}
}
