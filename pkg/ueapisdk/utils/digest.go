package utils

import (
	"crypto/md5"
	"encoding/hex"
)

const ENCODE = "UTF-8"

// MD5签名（带编码参数）
func SignMD5(aValue string, encoding string) string {
	h := md5.New()
	h.Write([]byte(aValue))
	return hex.EncodeToString(h.Sum(nil))
}
