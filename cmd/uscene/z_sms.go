package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"gorm.io/gorm"
)

type SMSCaptchaParam struct {
	Phone string `json:"phone"`
}

type SMSLoginParam struct {
	Phone      string `json:"phone"`
	Captcha    string `json:"captcha"`
	InviteCode string `json:"invite_code"`
}

type EmailCaptchaParam struct {
	Email string `json:"email"`
}

type EmailLoginParam struct {
	Email      string `json:"email"`
	Captcha    string `json:"captcha"`
	InviteCode string `json:"invite_code"`
}

const (
	testEmailBypassAddress = "123456@test.com"
	testEmailBypassCaptcha = "123456"
)

func isStagingBypassEmailLogin(env nucl.Env, email, captcha string) bool {
	return env == nucl.EnvStaging &&
		email == testEmailBypassAddress &&
		captcha == testEmailBypassCaptcha
}

// 发送验证码
func smsCaptcha(ec *middleware.AppRequestContext) error {
	req := SMSCaptchaParam{}
	if err := ec.Bind(&req); err != nil {
		sdlog.WithError(err).Error("smsCaptcha req err")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	sdlog.WithFields(sdlog.Fields{
		"user_id":      ec.AuthData.User.ID,
		"ip":           ec.RealIP(),
		"path":         ec.Request().URL.Path,
		"method":       ec.Request().Method,
		"request_body": req,
	}).Info("短信验证码请求开始")

	if req.Phone == "" {
		sdlog.WithError(sderr.Sentinel("手机号不能为空")).Error("smsCaptcha phone err")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("sms:lock:%s", req.Phone)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 生成验证码
	// code := sdrand.String(6, sdrand.Numbers)
	code := "123456"

	sendReq := &client.SendSmsRequest{
		PhoneNumbers:  tea.String(req.Phone),
		SignName:      tea.String(ec.Nu.Config.AliSMS.SignName),
		TemplateCode:  tea.String(ec.Nu.Config.AliSMS.TemplateCode),
		TemplateParam: tea.String(fmt.Sprintf(`{"code":"%s"}`, code)),
	}

	_, err := ec.Nu.AliSMSClient.SendSms(sendReq)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 将验证码存入 Redis，设置 5 分钟过期
	codeKey := fmt.Sprintf("sms:code:%s", req.Phone)
	err = ec.Nu.RedisClient.Set(ec.Request().Context(), codeKey, code, 5*time.Minute).Err()
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

// 手机号登录
func smsLogin(ec *middleware.AppRequestContext) error {
	req := SMSLoginParam{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.Phone == "" || req.Captcha == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 验证码校验
	codeKey := fmt.Sprintf("sms:code:%s", req.Phone)
	code, err := ec.Nu.RedisClient.Get(ec.Request().Context(), codeKey).Result()
	if err != nil || code != req.Captcha {
		return webapi.Error(common.ErrCaptcha).Render(ec)
	}

	// 获取用户锁
	lockKey := common.ConfigProviderByPhone + req.Phone
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 查找或创建用户
	var uLoginInfo model.UserAuth
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", req.Phone, common.ConfigProviderByPhone).
		First(&uLoginInfo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			newUid := ec.Nu.Snowflake.GenerateID()
			username := req.Phone

			// 添加用户
			errCreated := service.AddUser(
				ec.Request().Context(),
				ec.Nu,
				newUid,
				common.ConfigProviderByPhone,
				username,
				"", // avatar_url
				req.InviteCode,
				"",        // wallet_address
				req.Phone, // external_id
			)
			if errCreated != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}

			// 获取新创建的用户认证信息
			err = ec.Nu.DB.WithContext(ec.Request().Context()).
				Where("user_id = ? AND provider = ?", newUid, common.ConfigProviderByPhone).
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

	// 删除验证码
	ec.Nu.RedisClient.Del(ec.Request().Context(), codeKey)

	return webapi.OK(token).Render(ec)
}

// 发送邮箱验证码
func emailCaptcha(ec *middleware.AppRequestContext) error {
	req := EmailCaptchaParam{}
	if err := ec.Bind(&req); err != nil {
		sdlog.WithError(err).Error("emailCaptcha req err")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 验证邮箱格式
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		sdlog.WithError(sderr.Sentinel("邮箱格式不正确")).Error("emailCaptcha email format err")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	sdlog.WithFields(sdlog.Fields{
		"user_id":      ec.AuthData.User.ID,
		"ip":           ec.RealIP(),
		"path":         ec.Request().URL.Path,
		"method":       ec.Request().Method,
		"request_body": req,
	}).Info("邮箱验证码请求开始")

	if req.Email == "" {
		sdlog.WithError(sderr.Sentinel("邮箱不能为空")).Error("emailCaptcha email err")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("email:lock:%s", req.Email)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 声明code变量
	var code string

	// 检查Redis中是否已存在未失效的验证码
	codeKey := fmt.Sprintf("email:code:%s", req.Email)
	existingCode, err := ec.Nu.RedisClient.Get(ec.Request().Context(), codeKey).Result()
	if err == nil && existingCode != "" {
		// 如果存在未失效的验证码，直接使用该验证码
		code = existingCode
	} else {
		// 生成新的验证码
		code = sdrand.String(6, sdrand.Numbers)
	}

	if ec.Nu.Env == nucl.EnvStaging && req.Email == testEmailBypassAddress {
		code = testEmailBypassCaptcha
	} else {
		// 发送邮件
		err = ec.Nu.SendEmail(req.Email, code)
		if err != nil {
			sdlog.WithError(err).Error("发送邮件失败")
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	// 将验证码存入 Redis，设置 5 分钟过期
	err = ec.Nu.RedisClient.Set(ec.Request().Context(), codeKey, code, 5*time.Minute).Err()
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

// 邮箱验证码登录
func emailLogin(ec *middleware.AppRequestContext) error {
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
	token, err := webapi.GenerateTokenString([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress, webapi.TokenTypeRefresh, webapi.RefreshTokenDuration)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 测试环境万能验证码不依赖 Redis 验证码，登录后不做删除
	if !isStagingBypassEmailLogin(ec.Nu.Env, req.Email, req.Captcha) {
		ec.Nu.RedisClient.Del(ec.Request().Context(), codeKey)
	}

	return webapi.OK(token).Render(ec)
}
