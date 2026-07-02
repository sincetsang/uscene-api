package sdload

import (
	"encoding/json"
	"net/url"

	"TMA/pkg/sderr"
	"github.com/BurntSushi/toml"
)

func Bytes(loc string) ([]byte, error) {
	var scheme string
	u, err := url.Parse(loc)
	if err != nil {
		scheme = ""
	} else {
		scheme = u.Scheme
	}
	l, ok := loaders[scheme]
	if !ok {
		return nil, sderr.Newf("unknown scheme: '%s'", loc)
	}
	data, err := l.LoadBytes(loc)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return data, nil
}

func String(loc string) (string, error) {
	data, err := Bytes(loc)
	if err != nil {
		return "", sderr.WithStack(err)
	}
	return string(data), nil
}

func Json(loc string, v interface{}) error {
	data, err := Bytes(loc)
	if err != nil {
		return sderr.WithStack(err)
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return sderr.WithStack(err)
	}
	return nil
}

func Toml(loc string, v interface{}) error {
	data, err := Bytes(loc)
	if err != nil {
		return sderr.WithStack(err)
	}
	err = toml.Unmarshal(data, v)
	if err != nil {
		return sderr.WithStack(err)
	}
	return nil
}
