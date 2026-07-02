package models

// V2版订单请求
type TCodeReq struct {
	UEReqbaseV2

	Code string `json:"code"` // 入驻码
}
