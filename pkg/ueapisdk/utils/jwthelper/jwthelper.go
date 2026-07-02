package jwthelper

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// 获取JWT过期时间
func GetExpiration(jwtToken string) time.Time {
	if jwtToken == "" {
		return time.Time{} // 返回零值时间
	}

	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return time.Time{}
	}

	payload := parts[1]
	// 处理base64url编码
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")
	
	// 添加padding
	if pad := len(payload) % 4; pad > 0 {
		payload += strings.Repeat("=", 4-pad)
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return time.Time{}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return time.Time{}
	}

	if exp, ok := claims["exp"].(float64); ok {
		return time.Unix(int64(exp), 0)
	}
	
	return time.Time{}
}