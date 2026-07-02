package sdjson

func init() {
	Register(Converter{
		Bool:   toBool,
		String: toString,
		Int:    toInt64,
		Uint:   toUint64,
		Float:  toFloat64,
		Object: toObject,
		Array:  toArray,
		Any:    toAny,
	})
}
