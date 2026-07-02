package sdjson

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestV(t *testing.T) {
	assert.Equal(t, "hello", V("hello").Interface())
	assert.Equal(t, nil, V(nil).Interface())
	assert.True(t, V(nil).IsNil())
}

// Get

func TestValue_Field(t *testing.T) {
	v, err := UnmarshalValueString(`{
		"k1": {
			"k2": 33,
			"k3": "mm",
			"k4": true
		}
	}`)
	assert.NoError(t, err)
	assert.Equal(t, 33, v.Gets("k1", "k2").AsInt(0))
	assert.Equal(t, "mm", v.Get("k1").Get("k3").AsString(""))
	assert.Equal(t, "true", v.Gets("k1", "k4").AsString(""))
	assert.Equal(t, "not_found", v.Get("k1").Get("k5").AsString("not_found"))
	assert.Equal(t, "not_found", v.Get("k2").AsString("not_found"))
}

func TestValue_At(t *testing.T) {
	v, err := UnmarshalValueString(`["a", 3, {"k1":"v1"}]`)
	assert.NoError(t, err)
	assert.Equal(t, "a", v.At(0).AsString(""))
	assert.Equal(t, "3", v.At(1).AsString(""))
	assert.Equal(t, "v1", v.At(2).Get("k1").AsString(""))
}

// ToXXX

func TestValue_ToBool(t *testing.T) {
	// bool
	newfr(V(true).ToBool()).with(t).noErr().equal(true)
	newfr(V(false).ToBool()).with(t).noErr().equal(false)

	// other
	newfr(V(0).ToBool()).with(t).hasErr()
	newfr(V("true").ToBool()).with(t).hasErr()
	newfr(V(1.3).ToBool()).with(t).hasErr()
}

func TestValue_ToString(t *testing.T) {
	// string
	newfr(V("xx").ToString()).with(t).noErr().equal("xx")

	// other
	newfr(V(0).ToString()).with(t).hasErr()
	newfr(V(true).ToString()).with(t).hasErr()
	newfr(V(1.0).ToString()).with(t).hasErr()
}

func TestValue_ToInt(t *testing.T) {
	// other
	newfr(V(nil).ToInt()).with(t).hasErr()
	newfr(V(true).ToInt()).with(t).hasErr()
	newfr(V("0").ToInt()).with(t).hasErr()
	newfr(V(Object{}).ToInt()).with(t).hasErr()
	newfr(V(Array{}).ToInt()).with(t).hasErr()

	// number
	newfr(V(3).ToInt()).with(t).noErr().equal(3)
	newfr(V(int8(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(int16(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(int32(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(int64(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(uint(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(uint8(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(uint16(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(uint32(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(uint64(3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(3.3).ToInt()).with(t).noErr().equal(3)
	newfr(V(float32(3.3)).ToInt()).with(t).noErr().equal(3)
	newfr(V(json.Number("3.3")).ToInt()).with(t).noErr().equal(3)
	newfr(V(json.Number("3")).ToInt()).with(t).noErr().equal(3)
}

func TestValue_ToUint(t *testing.T) {
	// other
	newfr(V(nil).ToUint()).with(t).hasErr()
	newfr(V(true).ToUint()).with(t).hasErr()
	newfr(V("0").ToUint()).with(t).hasErr()
	newfr(V(Object{}).ToUint()).with(t).hasErr()
	newfr(V(Array{}).ToUint()).with(t).hasErr()

	// number
	newfr(V(3).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(int8(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(int16(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(int32(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(int64(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(uint(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(uint8(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(uint16(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(uint32(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(uint64(3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(3.3).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(float32(3.3)).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(json.Number("3.3")).ToUint()).with(t).noErr().equal(uint(3))
	newfr(V(json.Number("3")).ToUint()).with(t).noErr().equal(uint(3))
}

func TestValue_ToFloat64(t *testing.T) {
	// other
	newfr(V(nil).ToUint()).with(t).hasErr()
	newfr(V(true).ToUint()).with(t).hasErr()
	newfr(V("0").ToUint()).with(t).hasErr()
	newfr(V(Object{}).ToUint()).with(t).hasErr()
	newfr(V(Array{}).ToUint()).with(t).hasErr()

	// number
	newfr(V(3).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(int8(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(int16(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(int32(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(int64(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(uint(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(uint8(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(uint16(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(uint32(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(uint64(3)).ToFloat64()).with(t).noErr().equal(3.0)
	newfr(V(3.3).ToFloat64()).with(t).noErr().equal(3.3)
	newfr(V(float32(3.3)).ToFloat64()).with(t).noErr().equalFloat64(3.3)
	newfr(V(json.Number("3.3")).ToFloat64()).with(t).noErr().equal(3.3)
	newfr(V(json.Number("3")).ToFloat64()).with(t).noErr().equal(3.0)
}

func TestValue_ToObject(t *testing.T) {
	// other
	newfr(V(nil).ToObject()).with(t).hasErr()
	newfr(V(true).ToObject()).with(t).hasErr()
	newfr(V("0").ToObject()).with(t).hasErr()
	newfr(V(0.1).ToObject()).with(t).hasErr()
	newfr(V(Array{}).ToObject()).with(t).hasErr()

	// Object
	newfr(V(Object{}).ToObject()).with(t).isObject()
	newfr(V(Object(nil)).ToObject()).with(t).isNil()
	newfr(V(map[string]interface{}{"k1": "v1"}).ToObject()).with(t).isObject().deepEqual(Object{"k1": "v1"})
	newfr(V(map[string]interface{}(nil)).ToObject()).with(t).isNil()
	newfr(V(map[string]int{"k1": 0}).ToObject()).with(t).isObject().deepEqual(Object{"k1": 0})
	newfr(V(map[string]bool(nil)).ToObject()).with(t).isNil()
}

func TestValue_ToArray(t *testing.T) {
	//// other
	newfr(V(nil).ToArray()).with(t).hasErr()
	newfr(V(true).ToArray()).with(t).hasErr()
	newfr(V("0").ToArray()).with(t).hasErr()
	newfr(V(0.1).ToArray()).with(t).hasErr()
	newfr(V(Object{}).ToArray()).with(t).hasErr()
	//
	//// Array
	newfr(V(Array{}).ToArray()).with(t).isArray()
	newfr(V(Array{"a", 1}).ToArray()).with(t).isArray().deepEqual(Array{"a", 1})
	newfr(V(Array(nil)).ToArray()).with(t).isNil()
	newfr(V([]interface{}{"a"}).ToArray()).with(t).isArray().deepEqual(Array{"a"})
	newfr(V([]interface{}(nil)).ToArray()).with(t).isNil()
	newfr(V([]string{"a"}).ToArray()).with(t).isArray().deepEqual(Array{"a"})
	newfr(V([3]int{33}).ToArray()).with(t).isArray().deepEqual(Array{33, 0, 0})
}

func TestValue_ToAny(t *testing.T) {
	type person struct {
		Name string `json:"name"`
	}
	var p1 person
	assert.NoError(t, V(Object{"name": "xx"}).To(&p1))
	assert.Equal(t, "xx", p1.Name)

	var p2 person
	assert.NoError(t, V(person{"yy"}).To(&p2))
	assert.Equal(t, "yy", p2.Name)

	var p3 person
	assert.NoError(t, V(person{"zz"}).To(&p3))
	assert.Equal(t, "zz", p3.Name)

	var p4 Object
	assert.NoError(t, V(person{Name: "oo"}).To(&p4))
	assert.True(t, reflect.DeepEqual(p4, Object{"name": "oo"}))
}

// TryXXX

func TestValue_TryBool(t *testing.T) {
	// bool
	newfr(V(true).TryBool()).with(t).noErr().equal(true)
	newfr(V(false).TryBool()).with(t).noErr().equal(false)

	// int
	newfr(V(0).TryBool()).with(t).noErr().equal(false)
	newfr(V(1).TryBool()).with(t).noErr().equal(true)
	newfr(V(2).TryBool()).with(t).noErr().equal(true)

	// uint
	// int
	newfr(V(uint(0)).TryBool()).with(t).noErr().equal(false)
	newfr(V(uint(1)).TryBool()).with(t).noErr().equal(true)
	newfr(V(uint(2)).TryBool()).with(t).noErr().equal(true)

	// string
	newfr(V("true").TryBool()).with(t).noErr().equal(true)
	newfr(V("false").TryBool()).with(t).noErr().equal(false)

	// float
	newfr(V(0.0).TryBool()).with(t).noErr().equal(false)
	newfr(V(1.0).TryBool()).with(t).noErr().equal(true)
	newfr(V(3.3).TryBool()).with(t).noErr().equal(true)

	// other
	newfr(V(Object{}).TryBool()).with(t).hasErr()
	newfr(V(Array{}).TryBool()).with(t).hasErr()
}

func TestValue_TryString(t *testing.T) {
	// bool
	newfr(V(true).TryString()).with(t).noErr().equal("true")
	newfr(V(false).TryString()).with(t).noErr().equal("false")

	// string
	newfr(V("xx").TryString()).with(t).noErr().equal("xx")

	// int
	newfr(V(-33).TryString()).with(t).noErr().equal("-33")

	// uint
	newfr(V(uint(33)).TryString()).with(t).noErr().equal("33")

	// float64
	newfr(V(3.3).TryString()).with(t).noErr().equal("3.3")

	// object
	newfr(V(Object{}).TryString()).with(t).hasErr()

	// array
	newfr(V(Array{}).TryString()).with(t).hasErr()
}

func TestValue_TryInt(t *testing.T) {
	// bool
	newfr(V(true).TryInt()).with(t).noErr().equal(1)
	newfr(V(false).TryInt()).with(t).noErr().equal(0)

	// string
	newfr(V("33").TryInt()).with(t).noErr().equal(33)

	// int
	newfr(V(-33).TryInt()).with(t).noErr().equal(-33)

	// uint
	newfr(V(uint(33)).TryInt()).with(t).noErr().equal(33)

	// float64
	newfr(V(3.3).TryInt()).with(t).noErr().equal(3)

	// object
	newfr(V(Object{}).TryInt()).with(t).hasErr()

	// array
	newfr(V(Array{}).TryInt()).with(t).hasErr()
}

func TestValue_TryUint(t *testing.T) {
	// bool
	newfr(V(true).TryUint()).with(t).noErr().equal(uint(1))
	newfr(V(false).TryUint()).with(t).noErr().equal(uint(0))

	// string
	newfr(V("33").TryUint()).with(t).noErr().equal(uint(33))

	// int
	x := -33
	newfr(V(-33).TryUint()).with(t).noErr().equal(uint(x))

	// uint
	newfr(V(uint(33)).TryUint()).with(t).noErr().equal(uint(33))

	// float64
	newfr(V(3.3).TryUint()).with(t).noErr().equal(uint(3))

	// object
	newfr(V(Object{}).TryUint()).with(t).hasErr()

	// array
	newfr(V(Array{}).TryUint()).with(t).hasErr()
}

func TestValue_TryFloat64(t *testing.T) {
	// bool
	newfr(V(true).TryFloat64()).with(t).noErr().equal(1.0)
	newfr(V(false).TryFloat64()).with(t).noErr().equal(0.0)

	// string
	newfr(V("33.3").TryFloat64()).with(t).noErr().equal(33.3)

	// int
	newfr(V(-33).TryFloat64()).with(t).noErr().equal(-33.0)

	// uint
	newfr(V(uint(33)).TryFloat64()).with(t).noErr().equal(33.0)

	// float64
	newfr(V(3.3).TryFloat64()).with(t).noErr().equal(3.3)

	// object
	newfr(V(Object{}).TryFloat64()).with(t).hasErr()

	// array
	newfr(V(Array{}).TryFloat64()).with(t).hasErr()
}

func TestValue_TryObject(t *testing.T) {
	type person struct {
		Name string `json:"name"`
	}
	newfr(V(&person{Name: "xx"}).TryObject()).with(t).noErr().deepEqual(Object{"name": "xx"})
	newfr(V(person{Name: "yy"}).TryObject()).with(t).noErr().deepEqual(Object{"name": "yy"})
}

type funcReturn struct {
	v   interface{}
	err error
	t   *testing.T
}

func newfr(v interface{}, err error) *funcReturn {
	return &funcReturn{v, err, nil}
}

func (fr *funcReturn) with(t *testing.T) *funcReturn {
	fr.t = t
	return fr
}

func (fr *funcReturn) hasErr() *funcReturn {
	assert.Error(fr.t, fr.err)
	return fr
}

func (fr *funcReturn) noErr() *funcReturn {
	assert.NoError(fr.t, fr.err)
	return fr
}

func (fr *funcReturn) equal(expected interface{}) *funcReturn {
	assert.Equal(fr.t, expected, fr.v)
	return fr
}

func (fr *funcReturn) equalFloat64(expected float64) *funcReturn {
	assert.IsType(fr.t, 0.0, fr.v)
	v1 := fr.v.(float64)
	assert.True(fr.t, math.Abs(expected-v1) < 0.0000001)
	return fr
}

func (fr *funcReturn) isObject() *funcReturn {
	assert.IsType(fr.t, Object(nil), fr.v)
	return fr
}

func (fr *funcReturn) isArray() *funcReturn {
	assert.IsType(fr.t, Array(nil), fr.v)
	return fr
}

func (fr *funcReturn) isNil() *funcReturn {
	assert.Nil(fr.t, fr.v)
	return fr
}

func (fr *funcReturn) deepEqual(expected interface{}) *funcReturn {
	assert.True(fr.t, reflect.DeepEqual(expected, fr.v))
	return fr
}
