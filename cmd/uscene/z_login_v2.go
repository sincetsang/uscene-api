package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

func googleLoginV2(ec *middleware.AppRequestContext) error {
	req := GoogleLoginParam{}
	if err := ec.Bind(&req); err != nil || req.LoginType == "" || req.IdToken == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 直接调用 Google 的 tokeninfo endpoint
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", req.IdToken))
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	defer resp.Body.Close()

	var tokenInfo GoogleTokenInfo
	if err = json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var provider string
	switch req.LoginType {
	case "android":
		expectedClientID := ec.Nu.Config.GoogleOAuth.Android.ClientID
		if tokenInfo.Aud != expectedClientID {
			return webapi.Error(common.ErrService).Render(ec)
		}
		provider = common.ConfigProviderByGoogleAndroid
	case "ios":
		expectedClientID := ec.Nu.Config.GoogleOAuth.IOS.ClientID
		if tokenInfo.Aud != expectedClientID {
			return webapi.Error(common.ErrService).Render(ec)
		}
		provider = common.ConfigProviderByGoogleIos
	case "web":
		expectedClientID := ec.Nu.Config.GoogleOAuth.Web.ClientID
		if tokenInfo.Aud != expectedClientID {
			return webapi.Error(common.ErrService).Render(ec)
		}
		provider = common.ConfigProviderByGoogleWeb
	default:
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 获取用户锁
	lockKey := provider + tokenInfo.UserId
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 查找或创建用户
	var uLoginInfo model.UserAuth
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", tokenInfo.UserId, provider).
		First(&uLoginInfo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			newUid := ec.Nu.Snowflake.GenerateID()
			username := strconv.FormatInt(newUid, 10)

			errCreated := service.AddUser(
				ec.Request().Context(),
				ec.Nu,
				newUid,
				provider,
				username,
				"",
				req.InviteCode,
				"",
				tokenInfo.UserId,
			)
			if errCreated != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}

			err = ec.Nu.DB.WithContext(ec.Request().Context()).
				Where("user_id = ? AND provider = ?", newUid, provider).
				First(&uLoginInfo).Error
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 生成 JWT token
	token, err := webapi.GenerateTokenPair([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}

func appleLoginV2(ec *middleware.AppRequestContext) error {
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
	token, err := webapi.GenerateTokenPair([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}

// 手机号密码登录
func passwordLoginV2(ec *middleware.AppRequestContext) error {
	req := PasswordLoginParam{}
	if err := ec.Bind(&req); err != nil || req.Phone == "" || req.Password == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 先通过手机号在 user_auth 表中查找用户ID
	var uAuth model.UserAuth
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", req.Phone, common.ConfigProviderByPhone).
		First(&uAuth).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrUserNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 再通过用户ID查找用户信息和密码
	var user model.User
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", uAuth.UserID).
		First(&user).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查是否设置了密码
	if user.Password == "" || !service.CheckPassword(req.Password, user.Password) {
		return webapi.Error(common.ErrPasswordIncorrect).Render(ec)
	}

	// 生成 JWT token
	token, err := webapi.GenerateTokenPair([]byte(ec.Nu.Config.WalletJwtSecret), user.UserID, uAuth.WalletAddress)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}

// 邮箱验证码登录
func emailLoginV2(ec *middleware.AppRequestContext) error {
	req := EmailLoginParam{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.Email == "" || req.Captcha == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 验证码校验
	codeKey := fmt.Sprintf("email:code:%s", req.Email)
	code, err := ec.Nu.RedisClient.Get(ec.Request().Context(), codeKey).Result()
	if !isStagingBypassEmailLogin(ec.Nu.Env, req.Email, req.Captcha) && (err != nil || code != req.Captcha) {
		return webapi.Error(common.ErrCaptcha).Render(ec)
	}

	// 获取用户锁
	lockKey := common.ConfigProviderByEmail + req.Email
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 查找或创建用户
	var uLoginInfo model.UserAuth
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", req.Email, common.ConfigProviderByEmail).
		First(&uLoginInfo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			newUid := ec.Nu.Snowflake.GenerateID()
			username := req.Email

			// 添加用户
			errCreated := service.AddUser(
				ec.Request().Context(),
				ec.Nu,
				newUid,
				common.ConfigProviderByEmail,
				username,
				"", // avatar_url
				req.InviteCode,
				"",        // wallet_address
				req.Email, // external_id
			)
			if errCreated != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}

			// 获取新创建的用户认证信息
			err = ec.Nu.DB.WithContext(ec.Request().Context()).
				Where("user_id = ? AND provider = ?", newUid, common.ConfigProviderByEmail).
				First(&uLoginInfo).Error
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 生成 JWT token
	token, err := webapi.GenerateTokenPair([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 测试环境万能验证码不依赖 Redis 验证码，登录后不做删除
	if !isStagingBypassEmailLogin(ec.Nu.Env, req.Email, req.Captcha) {
		ec.Nu.RedisClient.Del(ec.Request().Context(), codeKey)
	}

	return webapi.OK(token).Render(ec)
}
