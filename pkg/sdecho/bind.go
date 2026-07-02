package sdecho

import (
	"io/ioutil"

	"TMA/pkg/sderr"
	"TMA/pkg/sdjson"
)

func (ec Context) BindJsonValue() (interface{}, error) {
	data, err := ioutil.ReadAll(ec.Request().Body)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	r, err := sdjson.UnmarshalValue(data)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return r, nil
}
