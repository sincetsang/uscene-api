package models

// 交换密钥请求参数
type ConfigReq struct {
	UEReqbaseV2

	TmpAppId     string `json:"tmpappid"`     // App临时ID
	TmpPublicKey string `json:"tmppublickey"` // App临时公钥
}
