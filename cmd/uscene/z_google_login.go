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

	"gorm.io/gorm"
)

type GoogleLoginParam struct {
	LoginType  string `json:"login_type"`
	InviteCode string `json:"invite_code"`
	IdToken    string `json:"id_token"`
}

type GoogleTokenInfo struct {
	Aud           string `json:"aud"`
	UserId        string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (g *GoogleTokenInfo) UnmarshalJSON(data []byte) error {
	type Alias GoogleTokenInfo
	aux := &struct {
		EmailVerified string `json:"email_verified"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.EmailVerified = aux.EmailVerified == "true"
	return nil
}

func googleLogin(ec *middleware.AppRequestContext) error {
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
	token, err := webapi.GenerateTokenString([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress, webapi.TokenTypeRefresh, webapi.RefreshTokenDuration)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}
