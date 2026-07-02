package models

// 密钥配置信息
type ConfigDto struct {
	Serverpublickey string `json:"serverpublickey"` // 服务端公钥
	Aeskey         string `json:"aeskey"`          // AES密钥
}