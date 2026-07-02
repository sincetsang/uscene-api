package models

// 泛型响应基类V2
type UERespbaseV2[T any] struct {
	UERespbase        // 继承基础响应字段
	Data       T `json:"data"` // 业务数据
}