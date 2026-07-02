package sdecho

import (
	"TMA/pkg/sderr"
	"github.com/labstack/echo/v4"
)

type Context struct {
	echo.Context
}

func H(h func(Context) error) func(echo.Context) error {
	return func(ec0 echo.Context) error {
		return sderr.WithStack(
			h(Context{ec0}),
		)
	}
}

func E(eh func(error, Context)) func(error, echo.Context) {
	return func(err0 error, ec0 echo.Context) {
		eh(err0, Context{ec0})
	}
}
