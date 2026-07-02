package models

type UeConfigParam2 struct {
	Aeskey      string `json:"aesKey"`      // AES加密密钥
	Privatekey  string `json:"privateKey"`  // 服务端私钥
	Accesstoken string `json:"accessToken"` // 访问令牌
}
