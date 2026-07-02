package models

// 密钥交换信息
type SecretDto struct {
	Opennodecode string `json:"opennodecode"` // 开放节点编码
	Uenodecode   string `json:"uenodecode"`   // UE节点编码
}