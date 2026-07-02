package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
)

// RefreshTokenParam 刷新 Token 请求参数
type RefreshTokenParam struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 刷新 Token 响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token 过期时间（秒）
}

// refreshToken 刷新 Access Token
func refreshToken(ec *middleware.AppRequestContext) error {
	// 解析请求参数
	req := RefreshTokenParam{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 使用 Refresh Token 获取新的 Token 对
	tokenPair, err := webapi.RefreshAccessToken(ec.Nu.Config.WalletJwtSecret, req.RefreshToken)
	if err != nil {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	// 返回新的 Token 对
	return webapi.OK(RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}).Render(ec)
}
