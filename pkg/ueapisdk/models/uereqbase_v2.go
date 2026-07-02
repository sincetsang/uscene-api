package models

import "github.com/sulink/ueapisdk/utils/constants"

type UEReqbaseV2 struct {
	Version  string `json:"version"`  // 当前版本号
	Sid      int    `json:"sid"`      // APPID
	Clientid int    `json:"clientid"` // 客户端类型
	Lang     string `json:"lang"`     // 语言
	Hardid   string `json:"hardid"`   // 硬件设备号
}

func NewUEReqbaseV2() UEReqbaseV2 {
	return UEReqbaseV2{
		Version:  "1.0.0",
		Sid:      81126,
		Clientid: 0,
		Lang:     "zh-cn",
		Hardid:   constants.HARDID,
	}
}
