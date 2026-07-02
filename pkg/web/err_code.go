package web

import (
	"TMA/pkg/web/webapi"
)

var (
	ErrInternal = &webapi.Result{
		Code:   500,
		ErrMsg: "服务器内部错误",
	}
)
