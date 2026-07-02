package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type AppleLoginParam struct {
	IdToken    string `json:"id_token"`
	InviteCode string `json:"invite_code"`
}

type AppleTokenPayload struct {
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
	Sub           string `json:"sub"` // 用户的唯一标识符
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type AppleKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type AppleKeys struct {
	Keys []AppleKey `json:"keys"`
}

// 缓存 Apple 公钥
var (
	appleKeysCache     = make(map[string]*rsa.PublicKey)
	appleKeysCacheTime time.Time
	appleKeysMutex     sync.RWMutex
	cacheExpiration    = 24 * time.Hour
)

func getApplePublicKeys() (map[string]*rsa.PublicKey, error) {
	appleKeysMutex.RLock()
	if time.Since(appleKeysCacheTime) < cacheExpiration && len(appleKeysCache) > 0 {
		defer appleKeysMutex.RUnlock()
		return appleKeysCache, nil
	}
	appleKeysMutex.RUnlock()

	// 获取新的公钥
	appleKeysMutex.Lock()
	defer appleKeysMutex.Unlock()

	// 再次检查，防止其他 goroutine 已经更新了缓存
	if time.Since(appleKeysCacheTime) < cacheExpiration && len(appleKeysCache) > 0 {
		return appleKeysCache, nil
	}

	resp, err := http.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, fmt.Errorf("failed to get apple public keys: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var keys AppleKeys
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse apple keys: %v", err)
	}

	// 清除旧缓存
	appleKeysCache = make(map[string]*rsa.PublicKey)

	// 解析每个公钥
	for _, key := range keys.Keys {
		if key.Kty != "RSA" || key.Use != "sig" {
			continue
		}

		// 解码模数和指数
		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		// 转换为 big.Int
		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		// 创建 RSA 公钥
		publicKey := &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}

		appleKeysCache[key.Kid] = publicKey
	}

	appleKeysCacheTime = time.Now()
	return appleKeysCache, nil
}

func appleLogin(ec *middleware.AppRequestContext) error {
	req := AppleLoginParam{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.IdToken == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 验证 Apple ID token
	userInfo, err := verifyAppleToken(req.IdToken)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证 token 是否过期
	if time.Now().Unix() > userInfo.Exp {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证发行者是否为 Apple
	if userInfo.Iss != "https://appleid.apple.com" {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证 audience 是否匹配
	expectedAud := ec.Nu.Config.AppleOAuth.BundleID
	if userInfo.Aud != expectedAud {
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	// 获取用户锁，防止并发注册
	lockKey := common.ConfigProviderByApple + userInfo.Sub
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 查找或创建用户
	var uLoginInfo model.UserAuth
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", userInfo.Sub, common.ConfigProviderByApple).
		First(&uLoginInfo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			newUid := ec.Nu.Snowflake.GenerateID()
			username := userInfo.Email
			if strings.HasSuffix(username, "@privaterelay.appleid.com") {
				username = userInfo.Sub
			}

			// 添加用户
			errCreated := service.AddUser(
				ec.Request().Context(),
				ec.Nu,
				newUid,
				common.ConfigProviderByApple,
				username,
				"", // avatar_url
				req.InviteCode,
				"",           // wallet_address
				userInfo.Sub, // external_id
			)
			if errCreated != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}

			// 获取新创建的用户认证信息
			err = ec.Nu.DB.WithContext(ec.Request().Context()).
				Where("user_id = ? AND provider = ?", newUid, common.ConfigProviderByApple).
				First(&uLoginInfo).Error
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 生成 JWT token
	token, err := webapi.GenerateTokenString([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress, webapi.TokenTypeRefresh, webapi.RefreshTokenDuration)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}

func verifyAppleToken(token string) (*AppleTokenPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// 解码 header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token header: %v", err)
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse token header: %v", err)
	}

	// 获取公钥
	publicKeys, err := getApplePublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get public keys: %v", err)
	}

	publicKey, ok := publicKeys[header.Kid]
	if !ok {
		return nil, fmt.Errorf("no matching key found for kid: %s", header.Kid)
	}

	// 验证签名
	signedData := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	hash := sha256.Sum256([]byte(signedData))
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %v", err)
	}

	// 解码并验证 payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %v", err)
	}

	var tokenPayload AppleTokenPayload
	if err := json.Unmarshal(payload, &tokenPayload); err != nil {
		return nil, fmt.Errorf("failed to parse token payload: %v", err)
	}

	return &tokenPayload, nil
}
