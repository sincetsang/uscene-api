package sdecho

import (
	"io/ioutil"

	"TMA/pkg/sderr"
)

func (ec Context) ReadRequestBody() ([]byte, error) {
	reader := ec.Request().Body
	r, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return r, nil
}

func (ec Context) ReadRequestBodyAsStr() (string, error) {
	b, err := ec.ReadRequestBody()
	if err != nil {
		return "", sderr.WithStack(err)
	}
	return string(b), nil
}
