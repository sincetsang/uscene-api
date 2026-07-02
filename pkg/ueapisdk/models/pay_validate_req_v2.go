package models

// V2版支付校验请求
type PayValidateReqV2 struct {
	UEReqbaseV2
	
	Orderno    string      `json:"orderno"`    // 外部订单号
	Ordernoue  string      `json:"ordernoue"`  // UE订单号
	Paycode    string      `json:"payCode"`   // 收款标识
	Orderitems []OrderItem `json:"orderitems"` // 支付金额列表
}