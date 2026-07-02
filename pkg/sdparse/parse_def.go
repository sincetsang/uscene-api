package sdparse

import (
	"time"
)

func Int64Def(s string, def int64) int64 {
	r, err := Int64(s)
	if err != nil {
		return def
	}
	return r
}

func IntDef(s string, def int) int {
	r, err := Int(s)
	if err != nil {
		return def
	}
	return r
}

func Uint64Def(s string, def uint64) uint64 {
	r, err := Uint64(s)
	if err != nil {
		return def
	}
	return r
}

func UintDef(s string, def uint) uint {
	r, err := Uint(s)
	if err != nil {
		return def
	}
	return r
}

func Float64Def(s string, def float64) float64 {
	r, err := Float64(s)
	if err != nil {
		return def
	}
	return r
}

func BoolDef(s string, def bool) bool {
	r, err := Bool(s)
	if err != nil {
		return def
	}
	return r
}

func TimeDef(s string, def time.Time) time.Time {
	r, err := Time(s)
	if err != nil {
		return def
	}
	return r
}
