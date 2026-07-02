package sdload

import (
	"TMA/pkg/sderr"
)

func MustBytes(loc string) []byte {
	data, err := Bytes(loc)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return data
}

func MustString(loc string) string {
	s, err := String(loc)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return s
}

func MustJson(loc string, v interface{}) {
	err := Json(loc, v)
	if err != nil {
		panic(sderr.WithStack(err))
	}
}

func MustToml(loc string, v interface{}) {
	err := Toml(loc, v)
	if err != nil {
		panic(sderr.WithStack(err))
	}
}
