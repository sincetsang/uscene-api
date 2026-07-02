package models

// 支付订单请求
type CreateOrderReq struct {
	UEReqbaseV2
	
	Amount  float64 `json:"amount"`  // 支付金额
	Unit    int     `json:"unit"`    // 钱包单位
	Paycode string  `json:"paycode"` // 收款人标识
	Orderno string  `json:"orderno"` // 订单号
}