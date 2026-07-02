package webapi

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// Token 相关常量
	AccessTokenDuration  = 2 * time.Hour       // Access Token 有效期 2 小时
	RefreshTokenDuration = 30 * 24 * time.Hour // Refresh Token 有效期 7 天
	TokenTypeAccess      = "access"
	TokenTypeRefresh     = "refresh"
)

type Claims struct {
	UserID             int64  `json:"user_id"`
	LoginWalletAddress string `json:"login_wallet_address"`
	TokenType          string `json:"token_type"` // 用于区分 token 类型
	jwt.RegisteredClaims
}

// TokenPair 包含 Access Token 和 Refresh Token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token 过期时间（秒）
}

// GenerateTokenPair 生成 Access Token 和 Refresh Token
func GenerateTokenPair(jwtSecret []byte, userID int64, loginWalletAddress string) (*TokenPair, error) {
	// 生成 Access Token
	accessToken, err := GenerateTokenString(jwtSecret, userID, loginWalletAddress, TokenTypeAccess, AccessTokenDuration)
	if err != nil {
		return nil, err
	}

	// 生成 Refresh Token
	refreshToken, err := GenerateTokenString(jwtSecret, userID, loginWalletAddress, TokenTypeRefresh, RefreshTokenDuration)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
	}, nil
}

// GenerateTokenString 生成指定类型的 token 字符串
func GenerateTokenString(jwtSecret []byte, userID int64, loginWalletAddress, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	expirationTime := now.Add(duration).Add(8 * time.Hour)

	claims := Claims{
		UserID:             userID,
		LoginWalletAddress: loginWalletAddress,
		TokenType:          tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "POTATO",
			Subject:   "user-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并验证 token
func ParseToken(jwtSecret string, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshAccessToken 使用 Refresh Token 获取新的 Access Token
func RefreshAccessToken(jwtSecret string, refreshToken string) (*TokenPair, error) {
	// 验证 Refresh Token
	claims, err := ParseToken(jwtSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	// 检查 token 类型
	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("not a refresh token")
	}

	// 生成新的 Access Token
	accessToken, err := GenerateTokenString([]byte(jwtSecret), claims.UserID, claims.LoginWalletAddress, TokenTypeAccess, AccessTokenDuration)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // 保持原有的 Refresh Token
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
	}, nil
}

// ValidateAccessToken 验证 Access Token
func ValidateAccessToken(jwtSecret, tokenString string) (*Claims, error) {
	claims, err := ParseToken(jwtSecret, tokenString)
	if err != nil {
		return nil, err
	}

	// 检查 token 类型
	if claims.TokenType != TokenTypeAccess {
		return nil, fmt.Errorf("not an access token")
	}

	return claims, nil
}
