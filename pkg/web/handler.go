package web

import (
	"TMA/pkg/nucl"
	"TMA/pkg/sdecho"
	"TMA/pkg/sderr"

	"github.com/labstack/echo/v4"
)

type NucleusHandlerFunc func(sdecho.Context, *nucl.Nucleus) error

func NH(nu *nucl.Nucleus, h NucleusHandlerFunc) echo.HandlerFunc {
	return func(ec echo.Context) error {
		err := h(sdecho.Context{Context: ec}, nu)
		return sderr.WithStack(err)
	}
}
