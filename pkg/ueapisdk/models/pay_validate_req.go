package models

// 支付回调校验请求
type PayValidateReq struct {
	UEReqbaseV2
	
	Orderno   string  `json:"orderno"`   // 外部订单号
	Ordernoue string  `json:"ordernoue"`// UE订单号
	PayCode   string  `json:"payCode"`  // 收款标识
	Amount    float64 `json:"amount"`   // 支付金额
}