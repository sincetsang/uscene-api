package models

// V2版订单请求
type CreateOrderV2Req struct {
	UEReqbaseV2
	
	Paycode       string      `json:"paycode"`       // 收款人标识
	Orderno       string      `json:"orderno"`       // 订单号
	Subject       string      `json:"subject"`       // 商品名称
	Body          string      `json:"body"`          // 描述
	NoticeUrl     string      `json:"noticeUrl"`     // 通知URL
	BusinessTypeid int        `json:"businessTypeid"`// UE支付业务类型id
	OrderItems    []OrderItem `json:"orderItems"`    // 支付金额列表
}
