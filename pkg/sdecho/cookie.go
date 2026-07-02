package sdecho

import (
	"net/http"
	"strconv"

	"TMA/pkg/sdencoding"
	"TMA/pkg/sderr"
	"TMA/pkg/sdjson"
)

// string

func (ec Context) CookieStr(name, def string) string {
	v, err := ec.Cookie(name)
	if err != nil {
		return def
	}
	return v.Value
}

func (ec Context) SetCookieStr(name, val string, path string, maxAge int) {
	ec.SetCookie(&http.Cookie{
		Name:   name,
		Value:  val,
		Path:   path,
		MaxAge: maxAge,
	})
}

func (ec Context) DeleteCookie(name string, path string) {
	ec.SetCookieStr(name, "", path, -1)
}

// int

func (ec Context) CookieInt(name string, def int) int {
	s := ec.CookieStr(name, "")
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) SetCookieInt(name string, val int, path string, maxAge int) {
	ec.SetCookieStr(name, strconv.Itoa(val), path, maxAge)
}

// int64

func (ec Context) CookieInt64(name string, def int64) int64 {
	s := ec.CookieStr(name, "")
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) SetCookieInt64(name string, val int64, path string, maxAge int) {
	ec.SetCookieStr(name, strconv.FormatInt(val, 10), path, maxAge)
}

// json

func (ec Context) CookieJson(name string, v interface{}) error {
	base64Str := ec.CookieStr(name, "")
	jsonBytes, err := sdencoding.Base64Url.DecodeStr(base64Str)
	if err != nil {
		return err
	}
	err = sdjson.Unmarshal(jsonBytes, v)
	if err != nil {
		return sderr.WithStack(err)
	}
	return nil
}

func (ec Context) SetCookieJson(name string, val interface{}, path string, maxAge int) error {
	jsonBytes, err := sdjson.Marshal(val)
	if err != nil {
		return err
	}
	ec.SetCookieStr(name, sdencoding.Base64Url.EncodeStr(jsonBytes), path, maxAge)
	return nil
}

// json object

func (ec Context) CookieJsonObject(name string, def sdjson.Object) sdjson.Object {
	base64Str := ec.CookieStr(name, "")
	jsonBytes, err := sdencoding.Base64Url.DecodeStr(base64Str)
	if err != nil {
		return def
	}
	v, err := sdjson.UnmarshalValue(jsonBytes)
	if err != nil {
		return def
	}
	return v.AsObject(def)
}

func (ec Context) SetCookieJsonObject(name string, val sdjson.Object, path string, maxAge int) error {
	jsonBytes, err := sdjson.Marshal(val)
	if err != nil {
		return err
	}
	ec.SetCookieStr(name, sdencoding.Base64Url.EncodeStr(jsonBytes), path, maxAge)
	return nil
}
