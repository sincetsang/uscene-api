package models

// 登录响应信息
type LoginDto struct {
	Accesstoken     string `json:"accesstoken"`     // 访问令牌
	Refreshtoken    string `json:"refreshtoken"`    // 刷新令牌
	Serverpublickey string `json:"serverpublickey"` // 服务端公钥
	Aeskey          string `json:"aeskey"`          // 加密密钥
}