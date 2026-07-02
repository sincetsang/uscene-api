package models

// UE支付回调数据
type UePayCallDto struct {
	Charge   string `json:"charge"`   // 支付凭据
	Sign     string `json:"sign"`     // 签名
	Orderno  string `json:"orderno"`  // 订单号
}