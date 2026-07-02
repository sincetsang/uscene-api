package models

// 支付码信息
type ChargeDto struct {
	Paycode string `json:"paycode"` // 支付码
	Orderno string `json:"orderno"` // 订单号
}