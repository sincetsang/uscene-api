package webapi

import (
	"TMA/pkg/common"
	"TMA/pkg/sderr"
	"TMA/pkg/sdjson"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"reflect"
)

type Facade func(int, interface{}) (interface{}, error)

type Result struct {
	Code   int
	Data   interface{}
	ErrMsg string
	Facade Facade
	Fields map[string]interface{}
	err    error
}

func OK(data interface{}) *Result {
	return &Result{
		Code:   200,
		Data:   data,
		ErrMsg: "",
	}
}

func OKM(data interface{}) *Result {
	return &Result{
		Code:   0,
		Data:   data,
		ErrMsg: "",
	}
}

func ErrorXm(err error) *Result {
	return &Result{
		Code:   -101,
		ErrMsg: err.Error(),
	}
}

func Error(err error) *Result {
	result := &Result{
		Code:   common.GetErrCode(err),
		Data:   nil,
		ErrMsg: common.GetErrMessage(err),
	}
	if common.GetErrCode(err) == common.DefaultErrCode {
		result.err = err
	}

	return result
}

func ErrorM(errMsg string) *Result {
	return Error(sderr.New(errMsg))
}

func ErrorMf(format string, args ...interface{}) *Result {
	return Error(sderr.Newf(format, args...))
}

func Of(data interface{}, err error) *Result {
	if err != nil {
		return Error(err)
	} else {
		return OK(data)
	}
}

func ParseResult(jsonData []byte) (*Result, error) {
	root, err := sdjson.UnmarshalValue(jsonData)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	o, err := root.ToObject()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	var code int
	var data json.RawMessage
	var errMsg string
	var fields = map[string]interface{}{}
	code, err = o.Get("code").ToInt()
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	if code == 200 {
		err = o.Get("data").To(&data)
		if err != nil {
			return nil, sderr.WithStack(err)
		}
	} else {
		errMsg = o.Get("err_msg").AsString("")
	}
	for k, v := range o {
		if k != "code" && k != "err_msg" && k != "data" {
			fields[k] = v
		}
	}
	return &Result{
		Code:   code,
		Data:   data,
		ErrMsg: errMsg,
		Fields: fields,
	}, nil
}

func (r *Result) UnmarshalData(v interface{}) error {
	j, ok := r.Data.(json.RawMessage)
	if !ok {
		return sderr.New("unmarshal result data error")
	}
	err := sdjson.Unmarshal(j, v)
	if err != nil {
		return sderr.WithStack(err)
	}
	return nil
}

func (r *Result) DataAsJson(pretty bool) string {
	if r.Data == nil {
		return ""
	}
	var d interface{}
	switch r.Data.(type) {
	case []byte:
		_ = json.Unmarshal(r.Data.([]byte), &d)
	case json.RawMessage:
		_ = json.Unmarshal(r.Data.(json.RawMessage), &d)
	default:
		d = r.Data
	}
	if d == nil {
		return ""
	}
	if pretty {
		return sdjson.MarshalIndentStringDef(d, "", "  ", "")
	} else {
		return sdjson.MarshalStringDef(d, "")
	}
}

func (r *Result) IsOK() bool {
	return r.Code == 200
}

func (r *Result) WithField(k string, v interface{}) *Result {
	r.ensureFields()
	r.Fields[k] = v
	return r
}

func (r *Result) WithFields(fields map[string]interface{}) *Result {
	r.ensureFields()
	for k, v := range fields {
		r.Fields[k] = v
	}
	return r
}

func (r *Result) WithFacade(facade Facade) *Result {
	r.Facade = facade
	return r
}

func (r *Result) Render(ec echo.Context) error {
	// Render elements
	var facadeData interface{}
	if r.Facade != nil {
		dv := reflect.ValueOf(r.Data)
		dt := dv.Type()
		renderElem := func(i int, v interface{}) (interface{}, error) {
			if v == nil {
				return nil, nil
			}
			rr, err := r.Facade(i, v)
			if err != nil {
				return nil, err
			}
			return rr, nil
		}
		if dt.Kind() == reflect.Slice || dt.Kind() == reflect.Array {
			n := dv.Len()
			renderedElems := make([]interface{}, 0, n)
			for i := 0; i < n; i++ {
				elem := dv.Index(i)
				renderedElem, err := renderElem(i, elem.Interface())
				if err != nil {
					return sderr.WithStack(err)
				}
				renderedElems = append(renderedElems, renderedElem)
			}
			facadeData = renderedElems
		} else {
			renderedOne, err := renderElem(0, dv.Interface())
			if err != nil {
				return sderr.WithStack(err)
			}
			facadeData = renderedOne
		}

	} else {
		facadeData = r.Data
	}

	// go
	o := sdjson.Object{}
	if r.IsOK() {
		o["code"] = 200
		o["data"] = facadeData
	} else {
		o["code"] = r.Code
		o["err_msg"] = r.ErrMsg
	}
	if r.Fields != nil {
		for k, v := range r.Fields {
			o[k] = v
		}
	}
	err := r.render(200, o, ec)
	return sderr.WithStack(err)
}

func (r *Result) RenderXM(ec echo.Context) error {
	// Render elements
	var facadeData interface{}
	if r.Facade != nil {
		dv := reflect.ValueOf(r.Data)
		dt := dv.Type()
		renderElem := func(i int, v interface{}) (interface{}, error) {
			if v == nil {
				return nil, nil
			}
			rr, err := r.Facade(i, v)
			if err != nil {
				return nil, err
			}
			return rr, nil
		}
		if dt.Kind() == reflect.Slice || dt.Kind() == reflect.Array {
			n := dv.Len()
			renderedElems := make([]interface{}, 0, n)
			for i := 0; i < n; i++ {
				elem := dv.Index(i)
				renderedElem, err := renderElem(i, elem.Interface())
				if err != nil {
					return sderr.WithStack(err)
				}
				renderedElems = append(renderedElems, renderedElem)
			}
			facadeData = renderedElems
		} else {
			renderedOne, err := renderElem(0, dv.Interface())
			if err != nil {
				return sderr.WithStack(err)
			}
			facadeData = renderedOne
		}

	} else {
		facadeData = r.Data
	}

	// go
	var err error
	o := sdjson.Object{}
	if r.Code == 0 {
		err = r.render(200, facadeData, ec)
	} else {
		o["code"] = r.Code
		o["description"] = r.ErrMsg
		err = r.render(200, o, ec)
	}
	if r.Fields != nil {
		for k, v := range r.Fields {
			o[k] = v
		}
	}
	return sderr.WithStack(err)
}

func (r *Result) ensureFields() {
	if r.Fields == nil {
		r.Fields = map[string]interface{}{}
	}
}

func (r *Result) render(code int, o interface{}, ec echo.Context) error {
	const jsonpCallbackFlag = "callback"
	jsonpCallback := ec.QueryParam("callback")
	var err error
	if jsonpCallback == jsonpCallbackFlag {
		err = ec.JSONP(code, jsonpCallback, o)
	} else {
		err = ec.JSON(code, o)
	}
	return sderr.WithStack(err)
}
