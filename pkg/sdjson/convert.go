package sdjson

import (
	"encoding/json"
	"reflect"
	"strconv"

	"TMA/pkg/sderr"
)

type BoolConv func(v interface{}, as bool) (bool, error)
type StringConv func(v interface{}, as bool) (string, error)
type IntConv func(v interface{}, as bool) (int64, error)
type UintConv func(v interface{}, as bool) (uint64, error)
type FloatConv func(v interface{}, as bool) (float64, error)
type ObjectConv func(v interface{}, as bool) (Object, error)
type ArrayConv func(v interface{}, as bool) (Array, error)
type AnyConv func(v, destPtr interface{}, as bool) error

type Converter struct {
	Bool   BoolConv
	String StringConv
	Int    IntConv
	Uint   UintConv
	Float  FloatConv
	Object ObjectConv
	Array  ArrayConv
	Any    AnyConv
}

var ErrIncompatibleType = sderr.Sentinel("incompatible type for convert value")

var (
	cc []*Converter
	gc Converter
)

func Register(c Converter) {
	cc = append(cc, &c)
	gc = mergeConverter(cc)
}

func mergeConverter(cc []*Converter) Converter {
	var bools []BoolConv
	var strs []StringConv
	var ints []IntConv
	var uints []UintConv
	var floats []FloatConv
	var objs []ObjectConv
	var arrs []ArrayConv
	var anys []AnyConv

	for _, c := range cc {
		if c.Bool != nil {
			bools = append(bools, c.Bool)
		}
		if c.String != nil {
			strs = append(strs, c.String)
		}
		if c.Int != nil {
			ints = append(ints, c.Int)
		}
		if c.Uint != nil {
			uints = append(uints, c.Uint)
		}
		if c.Float != nil {
			floats = append(floats, c.Float)
		}
		if c.Object != nil {
			objs = append(objs, c.Object)
		}
		if c.Array != nil {
			arrs = append(arrs, c.Array)
		}
		if c.Any != nil {
			anys = append(anys, c.Any)
		}
	}

	isIncompatible := func(err error) bool {
		return sderr.Is(err, ErrIncompatibleType)
	}

	var c Converter

	// bool
	if len(bools) == 1 {
		c.Bool = bools[0]
	} else if len(bools) > 1 {
		c.Bool = func(v interface{}, as bool) (bool, error) {
			for _, f := range bools {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return false, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return false, ErrIncompatibleType
		}
	}

	// string
	if len(strs) == 1 {
		c.String = strs[0]
	} else if len(strs) > 1 {
		c.String = func(v interface{}, as bool) (string, error) {
			for _, f := range strs {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return "", sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return "", ErrIncompatibleType
		}
	}

	// int
	if len(ints) == 1 {
		c.Int = ints[0]
	} else if len(ints) > 1 {
		c.Int = func(v interface{}, as bool) (int64, error) {
			for _, f := range ints {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return 0, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return 0, ErrIncompatibleType
		}
	}

	// uint
	if len(uints) == 1 {
		c.Uint = uints[0]
	} else if len(uints) > 1 {
		c.Uint = func(v interface{}, as bool) (uint64, error) {
			for _, f := range uints {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return 0, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return 0, ErrIncompatibleType
		}
	}

	// float
	if len(floats) == 1 {
		c.Float = floats[0]
	} else if len(floats) > 1 {
		c.Float = func(v interface{}, as bool) (float64, error) {
			for _, f := range floats {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return 0.0, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return 0.0, ErrIncompatibleType
		}
	}

	// object
	if len(objs) == 1 {
		c.Object = objs[0]
	} else if len(objs) > 1 {
		c.Object = func(v interface{}, as bool) (Object, error) {
			for _, f := range objs {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return nil, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return nil, ErrIncompatibleType
		}
	}

	// any
	if len(arrs) == 1 {
		c.Array = arrs[0]
	} else if len(arrs) > 1 {
		c.Array = func(v interface{}, as bool) (Array, error) {
			for _, f := range arrs {
				r, err := f(v, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return nil, sderr.WithStack(err)
					}
				} else {
					return r, nil
				}
			}
			return nil, ErrIncompatibleType
		}
	}

	// any
	if len(anys) == 1 {
		c.Any = anys[0]
	} else if len(anys) > 1 {
		c.Any = func(v, destPtr interface{}, as bool) error {
			for _, f := range anys {
				err := f(v, destPtr, as)
				if err != nil {
					if isIncompatible(err) {
						continue
					} else {
						return sderr.WithStack(err)
					}
				} else {
					return nil
				}
			}
			return ErrIncompatibleType
		}
	}

	return c
}

func toBool(v interface{}, as bool) (bool, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return false, ErrIncompatibleType
	}

	// bool
	if v1, ok := v.(bool); ok {
		return v1, nil
	}
	if as {
		switch v1 := v.(type) {
		// string
		case string:
			b, err := strconv.ParseBool(v1)
			if err != nil {
				return false, sderr.WithStack(err)
			}
			return b, nil

		// number
		case json.Number:
			if isFloat(v1) {
				if v2, err := v1.Float64(); err != nil {
					return false, sderr.WithStack(err)
				} else {
					return v2 != 0.0, nil
				}
			} else {
				if v2, err := v1.Int64(); err != nil {
					return false, sderr.WithStack(err)
				} else {
					return v2 != 0, nil
				}
			}
		case int:
			return v1 != 0, nil
		case int64:
			return v1 != 0, nil
		case uint:
			return v1 != 0, nil
		case uint64:
			return v1 != 0, nil
		case float64:
			return v1 != 0.0, nil
		case int8:
			return v1 != 0, nil
		case int16:
			return v1 != 0, nil
		case int32:
			return v1 != 0, nil
		case uint8:
			return v1 != 0, nil
		case uint16:
			return v1 != 0, nil
		case uint32:
			return v1 != 0, nil
		case float32:
			return v1 != 0.0, nil
		}
	}
	return false, ErrIncompatibleType
}

func toString(v interface{}, as bool) (string, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return "", ErrIncompatibleType
	}
	// string
	if v1, ok := v.(string); ok {
		return v1, nil
	}

	if as {
		switch v1 := v.(type) {
		// bool
		case bool:
			return strconv.FormatBool(v1), nil
		// number
		case json.Number:
			return v1.String(), nil
		case int:
			return strconv.FormatInt(int64(v1), 10), nil
		case int64:
			return strconv.FormatInt(v1, 10), nil
		case uint:
			return strconv.FormatUint(uint64(v1), 10), nil
		case uint64:
			return strconv.FormatUint(v1, 10), nil
		case float64:
			return strconv.FormatFloat(v1, 'f', -1, 64), nil
		case int8:
			return strconv.FormatInt(int64(v1), 10), nil
		case int16:
			return strconv.FormatInt(int64(v1), 10), nil
		case int32:
			return strconv.FormatInt(int64(v1), 10), nil
		case uint8:
			return strconv.FormatUint(uint64(v1), 10), nil
		case uint16:
			return strconv.FormatUint(uint64(v1), 10), nil
		case uint32:
			return strconv.FormatUint(uint64(v1), 10), nil
		case float32:
			return strconv.FormatFloat(float64(v1), 'f', -1, 32), nil
		}
	}

	return "", ErrIncompatibleType
}

func toInt64(v interface{}, as bool) (int64, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return 0, ErrIncompatibleType
	}

	// number
	switch v1 := v.(type) {
	case json.Number:
		if isFloat(v1) {
			if v2, err := v1.Float64(); err != nil {
				return 0, sderr.WithStack(err)
			} else {
				return int64(v2), nil
			}
		} else {
			if v2, err := v1.Int64(); err != nil {
				return 0, sderr.WithStack(err)
			} else {
				return v2, nil
			}
		}
	case int:
		return int64(v1), nil
	case int64:
		return v1, nil
	case uint:
		return int64(v1), nil
	case uint64:
		return int64(v1), nil
	case float64:
		return int64(v1), nil
	case int8:
		return int64(v1), nil
	case int16:
		return int64(v1), nil
	case int32:
		return int64(v1), nil
	case uint8:
		return int64(v1), nil
	case uint16:
		return int64(v1), nil
	case uint32:
		return int64(v1), nil
	case float32:
		return int64(v1), nil
	}

	if as {
		switch v1 := v.(type) {
		// bool
		case bool:
			if v1 {
				return 1, nil
			} else {
				return 0, nil
			}
		// string
		case string:
			if isFloat(json.Number(v1)) {
				if v2, err := json.Number(v1).Float64(); err != nil {
					return 0, sderr.WithStack(err)
				} else {
					return int64(v2), nil
				}
			} else {
				if v2, err := json.Number(v1).Int64(); err != nil {
					return 0, sderr.WithStack(err)
				} else {
					return v2, nil
				}
			}
		}
	}

	return 0, ErrIncompatibleType
}

func toUint64(v interface{}, as bool) (uint64, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return 0, ErrIncompatibleType
	}

	// number
	switch v1 := v.(type) {
	case json.Number:
		if isFloat(v1) {
			if v2, err := v1.Float64(); err != nil {
				return 0, sderr.WithStack(err)
			} else {
				return uint64(v2), nil
			}
		} else {
			if v2, err := strconv.ParseUint(string(v1), 10, 64); err != nil {
				return 0, sderr.WithStack(err)
			} else {
				return v2, nil
			}
		}
	case int:
		return uint64(v1), nil
	case int64:
		return uint64(v1), nil
	case uint:
		return uint64(v1), nil
	case uint64:
		return v1, nil
	case float64:
		return uint64(v1), nil
	case int8:
		return uint64(v1), nil
	case int16:
		return uint64(v1), nil
	case int32:
		return uint64(v1), nil
	case uint8:
		return uint64(v1), nil
	case uint16:
		return uint64(v1), nil
	case uint32:
		return uint64(v1), nil
	case float32:
		return uint64(v1), nil
	}

	if as {
		switch v1 := v.(type) {
		// bool
		case bool:
			if v1 {
				return 1, nil
			} else {
				return 0, nil
			}
		// string
		case string:
			if isFloat(json.Number(v1)) {
				if v2, err := json.Number(v1).Float64(); err != nil {
					return 0, sderr.WithStack(err)
				} else {
					return uint64(v2), nil
				}
			} else {
				if v2, err := strconv.ParseUint(v1, 10, 64); err != nil {
					return 0, sderr.WithStack(err)
				} else {
					return v2, nil
				}
			}
		}
	}

	return 0, ErrIncompatibleType
}

func toFloat64(v interface{}, as bool) (float64, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return 0, ErrIncompatibleType
	}

	// number
	switch v1 := v.(type) {
	case json.Number:
		if v2, err := v1.Float64(); err != nil {
			return 0, sderr.WithStack(err)
		} else {
			return v2, nil
		}
	case float64:
		return v1, nil
	case float32:
		return float64(v1), nil
	case int:
		return float64(v1), nil
	case int64:
		return float64(v1), nil
	case uint:
		return float64(v1), nil
	case uint64:
		return float64(v1), nil
	case int8:
		return float64(v1), nil
	case int16:
		return float64(v1), nil
	case int32:
		return float64(v1), nil
	case uint8:
		return float64(v1), nil
	case uint16:
		return float64(v1), nil
	case uint32:
		return float64(v1), nil
	}

	if as {
		switch v1 := v.(type) {
		// bool
		case bool:
			if v1 {
				return 1.0, nil
			} else {
				return 0.0, nil
			}
		// string
		case string:
			if v2, err := json.Number(v1).Float64(); err != nil {
				return 0.0, sderr.WithStack(err)
			} else {
				return v2, nil
			}
		}
	}

	return 0.0, ErrIncompatibleType
}

func toObject(v interface{}, as bool) (Object, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return nil, ErrIncompatibleType
	}

	// map like
	if v1, ok := v.(map[string]interface{}); ok {
		return v1, nil
	} else if v1, ok := v.(Object); ok {
		return v1, nil
	} else if rv := reflect.ValueOf(v); rv.Type().Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		return genericMapToObject(rv)
	}

	if as {
		// struct
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		if rt.Kind() == reflect.Struct {
			return structToObject(rv)
		} else if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
			return structToObject(rv)
		}
	}

	return nil, ErrIncompatibleType
}

func toArray(v interface{}, _ bool) (Array, error) {
	v = unbox(v)

	// nil
	if v == nil {
		return nil, ErrIncompatibleType
	}

	// slice like
	if v1, ok := v.([]interface{}); ok {
		return v1, nil
	} else if v1, ok := v.(Array); ok {
		return v1, nil
	} else if rv := reflect.ValueOf(v); rv.Type().Kind() == reflect.Slice || rv.Type().Kind() == reflect.Array {
		return genericSliceToArray(rv)
	}

	return nil, ErrIncompatibleType
}

func toAny(v, destPtr interface{}, _ bool) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return sderr.WithStack(err)
	}
	err = json.Unmarshal(raw, destPtr)
	if err != nil {
		return sderr.WithStack(err)
	}
	return nil
}

func genericMapToObject(rv reflect.Value) (map[string]interface{}, error) {
	if rv.IsNil() {
		return nil, nil
	}
	l := rv.Len()
	m := make(map[string]interface{}, l)
	iter := rv.MapRange()
	for iter.Next() {
		k := iter.Key().Interface().(string)
		v := iter.Value().Interface()
		m[k] = v
	}
	return m, nil
}

func structToObject(rv reflect.Value) (map[string]interface{}, error) {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, nil
	}
	raw, err := json.Marshal(rv.Interface())
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	var m map[string]interface{}
	err = json.Unmarshal(raw, &m)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return m, nil
}

func genericSliceToArray(rv reflect.Value) ([]interface{}, error) {
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return nil, nil
	}
	l := rv.Len()
	a := make([]interface{}, 0, l)
	for i := 0; i < l; i++ {
		a = append(a, rv.Index(i).Interface())
	}
	return a, nil
}
