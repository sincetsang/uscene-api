package models

type UeConfigParam struct {
	Aeskey     string            `json:"aesKey"`     // AES加密密钥
	Privatekey string            `json:"privateKey"` // 服务端私钥
	Header     map[string]string `json:"header"`     // 请求头信息
}
