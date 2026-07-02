package models

// 登录请求参数
type ApiLoginReq struct {
	UEReqbaseV2 // 嵌入基础请求结构
	
	Accesskeyid  string `json:"accesskeyid"`   // 业务分配的accesskeyid
	Accesssecret string `json:"accesssecret"`  // 业务分配的accesssecret
	PublicKey    string `json:"publickey"`     // 公钥
}