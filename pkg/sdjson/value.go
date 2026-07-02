package sdjson

import (
	"encoding/json"

	"TMA/pkg/sderr"
)

type Value struct {
	v interface{}
}

func V(v interface{}) Value {
	return Value{unbox(v)}
}

func unbox(v interface{}) interface{} {
	switch v1 := v.(type) {
	case nil:
		return nil
	case Value:
		return v1.v
	case *Value:
		if v1 == nil {
			return nil
		} else {
			return v1.v
		}
	default:
		return v
	}
}

// IsXXX

func (v Value) IsNil() bool {
	return v.v == nil
}

// Get

func (v Value) Get(k string) Value {
	return v.AsObject(nil).Get(k)
}

func (v Value) Gets(k string, subKeys ...string) Value {
	if len(subKeys) <= 0 {
		return v.Get(k)
	} else {
		r := v.Get(k)
		for _, subKey := range subKeys {
			r = r.Get(subKey)
		}
		return r
	}
}

// At

func (v Value) At(i int) Value {
	return v.AsArray(nil).At(i)
}

// marshal

func (v Value) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(v.v)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return raw, nil
}

func (v *Value) UnmarshalJSON(raw []byte) error {
	var v0 interface{}
	err := json.Unmarshal(raw, &v0)
	if err != nil {
		return sderr.WithStack(err)
	}
	v.v = v0
	return nil
}

// interface{}

func (v Value) Interface() interface{} {
	return v.v
}

// To

func (v Value) ToBool() (bool, error) {
	if v1, err := gc.Bool(v.v, false); err != nil {
		return false, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToString() (string, error) {
	if v1, err := gc.String(v.v, false); err != nil {
		return "", sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToInt() (int, error) {
	if v1, err := gc.Int(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int(v1), nil
	}
}

func (v Value) ToInt8() (int8, error) {
	if v1, err := gc.Int(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int8(v1), nil
	}
}

func (v Value) ToInt16() (int16, error) {
	if v1, err := gc.Int(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int16(v1), nil
	}
}

func (v Value) ToInt32() (int32, error) {
	if v1, err := gc.Int(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int32(v1), nil
	}
}

func (v Value) ToInt64() (int64, error) {
	if v1, err := gc.Int(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToUint() (uint, error) {
	if v1, err := gc.Uint(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint(v1), nil
	}
}

func (v Value) ToUint8() (uint8, error) {
	if v1, err := gc.Uint(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint8(v1), nil
	}
}

func (v Value) ToUint16() (uint16, error) {
	if v1, err := gc.Uint(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint16(v1), nil
	}
}

func (v Value) ToUint32() (uint32, error) {
	if v1, err := gc.Uint(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint32(v1), nil
	}
}

func (v Value) ToUint64() (uint64, error) {
	if v1, err := gc.Uint(v.v, false); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToFloat64() (float64, error) {
	if v1, err := gc.Float(v.v, false); err != nil {
		return 0.0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToFloat32() (float32, error) {
	if v1, err := gc.Float(v.v, false); err != nil {
		return 0.0, sderr.WithStack(err)
	} else {
		return float32(v1), nil
	}
}

func (v Value) ToObject() (Object, error) {
	if v1, err := gc.Object(v.v, false); err != nil {
		return nil, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) ToArray() (Array, error) {
	if v1, err := gc.Array(v.v, false); err != nil {
		return nil, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) To(destPtr interface{}) error {
	if err := gc.Any(v.v, destPtr, false); err != nil {
		return sderr.WithStack(err)
	}
	return nil
}

// Try

func (v Value) TryBool() (bool, error) {
	if v1, err := gc.Bool(v.v, true); err != nil {
		return false, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryString() (string, error) {
	if v1, err := gc.String(v.v, true); err != nil {
		return "", sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryInt() (int, error) {
	if v1, err := gc.Int(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int(v1), nil
	}
}

func (v Value) TryInt8() (int8, error) {
	if v1, err := gc.Int(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int8(v1), nil
	}
}

func (v Value) TryInt16() (int16, error) {
	if v1, err := gc.Int(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int16(v1), nil
	}
}

func (v Value) TryInt32() (int32, error) {
	if v1, err := gc.Int(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return int32(v1), nil
	}
}

func (v Value) TryInt64() (int64, error) {
	if v1, err := gc.Int(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryUint() (uint, error) {
	if v1, err := gc.Uint(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint(v1), nil
	}
}

func (v Value) TryUint8() (uint8, error) {
	if v1, err := gc.Uint(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint8(v1), nil
	}
}

func (v Value) TryUint16() (uint16, error) {
	if v1, err := gc.Uint(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint16(v1), nil
	}
}

func (v Value) TryUint32() (uint32, error) {
	if v1, err := gc.Uint(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return uint32(v1), nil
	}
}

func (v Value) TryUint64() (uint64, error) {
	if v1, err := gc.Uint(v.v, true); err != nil {
		return 0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryFloat64() (float64, error) {
	if v1, err := gc.Float(v.v, true); err != nil {
		return 0.0, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryFloat32() (float32, error) {
	if v1, err := gc.Float(v.v, true); err != nil {
		return 0.0, sderr.WithStack(err)
	} else {
		return float32(v1), nil
	}
}

func (v Value) TryObject() (Object, error) {
	if v1, err := gc.Object(v.v, true); err != nil {
		return nil, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) TryArray() (Array, error) {
	if v1, err := gc.Array(v.v, true); err != nil {
		return nil, sderr.WithStack(err)
	} else {
		return v1, nil
	}
}

func (v Value) Try(destPtr interface{}) error {
	return v.To(destPtr)
}

// As

func (v Value) AsBool(def bool) bool {
	if v1, err := v.TryBool(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsString(def string) string {
	if v1, err := v.TryString(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsInt(def int) int {
	if v1, err := v.TryInt(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsInt8(def int8) int8 {
	if v1, err := v.TryInt8(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsInt16(def int16) int16 {
	if v1, err := v.TryInt16(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsInt32(def int32) int32 {
	if v1, err := v.TryInt32(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsInt64(def int64) int64 {
	if v1, err := v.TryInt64(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsUint(def uint) uint {
	if v1, err := v.TryUint(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsUint8(def uint8) uint8 {
	if v1, err := v.TryUint8(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsUint16(def uint16) uint16 {
	if v1, err := v.TryUint16(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsUint32(def uint32) uint32 {
	if v1, err := v.TryUint32(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsUint64(def uint64) uint64 {
	if v1, err := v.TryUint64(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsFloat64(def float64) float64 {
	if v1, err := v.TryFloat64(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsObject(def Object) Object {
	if v1, err := v.TryObject(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) AsArray(def Array) Array {
	if v1, err := v.TryArray(); err != nil {
		return def
	} else {
		return v1
	}
}

func (v Value) As(destPtr interface{}, def interface{}) interface{} {
	if err := v.Try(destPtr); err != nil {
		return def
	} else {
		return destPtr
	}
}
