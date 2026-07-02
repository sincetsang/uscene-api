package sdparse

import (
	"time"
)

func MustInt64(s string) int64 {
	r, err := Int64(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustInt(s string) int {
	r, err := Int(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustUint64(s string) uint64 {
	r, err := Uint64(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustUint(s string) uint {
	r, err := Uint(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustFloat64(s string) float64 {
	r, err := Float64(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustBool(s string) bool {
	r, err := Bool(s)
	if err != nil {
		panic(err)
	}
	return r
}

func MustTime(s string) time.Time {
	r, err := Time(s)
	if err != nil {
		panic(err)
	}
	return r
}
