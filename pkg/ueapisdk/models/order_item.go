package models

// 支付订单项（与CreateOrderV2Req中的OrderItem合并）
type OrderItem struct {
	Paytype int     `json:"paytype"` // 支付类型：1-DOS,3-CNV,25-FEC
	Amount  float64 `json:"amount"`  // 消费金额
	Unit    int     `json:"unit"`    // 钱包单位
}