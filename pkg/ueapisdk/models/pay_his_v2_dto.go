package models

import "time"

// 支付历史记录
type PayHisV2Dto struct {
	Ordernoue  string    `json:"ordernoue"`  // UE订单号
	Optype     string    `json:"optype"`     // 操作类型
	Paycode    string    `json:"paycode"`    // 收款人标识
	Orderno    string    `json:"orderno"`    // 订单号
	Subject    string    `json:"subject"`    // 商品名称
	Body       string    `json:"body"`       // 描述
	Createtime time.Time `json:"createtime"` // 支付时间
	Paystatus  string    `json:"paystatus"`  // 支付状态
	Orderitems string    `json:"orderitems"` // 支付金额列表（JSON）
}
