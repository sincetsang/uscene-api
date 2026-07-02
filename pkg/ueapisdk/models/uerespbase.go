package models

// 响应基础结构
type UERespbase struct {
	// 返回值，正数为成功，0或负数表示失败
	Result int `json:"result"`
	// 返回值的中文描述
	Message string `json:"message"`
}
