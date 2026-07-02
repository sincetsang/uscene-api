package nucl

import (
	"TMA/pkg/sdlog"
	"fmt"
	"sync"
	"time"

	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"github.com/sulink/ueapisdk/utils"
	"github.com/sulink/ueapisdk/utils/constants"
	"github.com/sulink/ueapisdk/utils/jwthelper"
	"github.com/sulink/ueapisdk/utils/rsa"
)

var (
	loginLock sync.Mutex
)

// InitUE 初始化UE API配置
func (n *Nucleus) InitUE() error {
	return n.withLog("UE", func() error {
		ue_api_v2.Initialization(n.Config.UE.UeApiUrl)
		n.UeParam = make(map[string]interface{})
		err := n.LoginCore()
		if err != nil {
			return err
		}
		// 启动定时检查
		go n.startTokenCheck()
		return nil
	})
}

// startTokenCheck 启动定时检查token
func (n *Nucleus) startTokenCheck() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := n.LoginRefresh(); err != nil {
				fmt.Printf("刷新token失败: %v\n", err)
			}
		}
	}
}

// LoginCore 执行登录流程
func (n *Nucleus) LoginCore() error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("登录流程异常: %v\n", err)
		}
	}()

	// 1. 生成RSA密钥对
	keyPair, err := rsa.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("生成RSA密钥对失败: %v", err)
	}

	// 2. 获取配置
	configReq := models.ConfigReq{
		UEReqbaseV2:  models.NewUEReqbaseV2(),
		TmpAppId:     utils.GetRandomString(16),
		TmpPublicKey: rsa.PEMToBase64(keyPair[constants.RSA_PUBLICKEY]),
	}
	configResp, err := ue_api_v2.GetConfig(configReq)
	sdlog.Info("configResp %v", configResp)
	if err != nil {
		return fmt.Errorf("获取配置失败: %v", err)
	}

	if configResp.Result <= 0 {
		return fmt.Errorf("获取配置失败: %+v", configResp)
	}

	// 3. 解密AES密钥
	decryptedAesKey, err := rsa.Decrypt(configResp.Data.Aeskey, keyPair[constants.RSA_PRIVATEKEY])
	if err != nil {
		return fmt.Errorf("解密AES密钥失败: %v", err)
	}
	configResp.Data.Serverpublickey = rsa.Base64ToPEM(configResp.Data.Serverpublickey, true)

	// 4. 准备登录请求
	encryAccesssecret, err := rsa.Encrypt(n.Config.UE.AccessSecret, configResp.Data.Serverpublickey)
	if err != nil {
		return fmt.Errorf("加密AccessSecret失败: %v", err)
	}

	loginReq := models.ApiLoginReq{
		UEReqbaseV2:  models.NewUEReqbaseV2(),
		Accesskeyid:  n.Config.UE.AccessKeyId,
		Accesssecret: encryAccesssecret,
		PublicKey:    rsa.PEMToBase64(keyPair[constants.RSA_PUBLICKEY]),
	}

	// 5. 执行登录
	loginParam := models.UeConfigParam{
		Aeskey:     decryptedAesKey,
		Privatekey: keyPair[constants.RSA_PRIVATEKEY],
		Header:     map[string]string{constants.TMP_APP_ID: configReq.TmpAppId},
	}
	loginResp, err := ue_api_v2.Login(loginReq, loginParam)
	if err != nil {
		return fmt.Errorf("登录失败: %v", err)
	}
	if loginResp.Result <= 0 {
		return fmt.Errorf("登录失败: %+v", loginResp)
	}

	// 6. 保存配置
	n.UeParam[constants.ACCESS_TOKEN] = loginResp.Data.Accesstoken
	n.UeParam[constants.REFRESH_TOKEN] = loginResp.Data.Refreshtoken
	decryptedLoginAesKey, err := rsa.Decrypt(loginResp.Data.Aeskey, keyPair[constants.RSA_PRIVATEKEY])
	if err != nil {
		return fmt.Errorf("解密登录AES密钥失败: %v", err)
	}
	n.UeParam[constants.AES_KEY] = decryptedLoginAesKey
	n.UeParam[constants.SERVER_PUBLICKEY] = rsa.Base64ToPEM(loginResp.Data.Serverpublickey, true)
	n.UeParam[constants.RSA_PRIVATEKEY] = keyPair[constants.RSA_PRIVATEKEY]

	// 更新 UeConfigParam
	n.UeConfigParam = models.UeConfigParam2{
		Aeskey:      n.UeParam[constants.AES_KEY].(string),
		Privatekey:  n.UeParam[constants.RSA_PRIVATEKEY].(string),
		Accesstoken: n.UeParam[constants.ACCESS_TOKEN].(string),
	}

	fmt.Println("UE API登录成功，配置信息:")
	// jsonStr, _ := json.MarshalIndent(n.UeParam, "", "  ")
	// fmt.Println(string(jsonStr))

	return nil
}

// LoginRefresh 检查并刷新token
func (n *Nucleus) LoginRefresh() error {
	loginLock.Lock()
	defer loginLock.Unlock()
	if n.UeParam[constants.ACCESS_TOKEN] != nil && n.UeParam[constants.REFRESH_TOKEN] != nil {
		expToken := jwthelper.GetExpiration(n.UeParam[constants.ACCESS_TOKEN].(string))

		// 提前32分钟判断过期
		adjustedExpToken := expToken.Add(-32 * time.Minute)
		if time.Now().Before(adjustedExpToken) {
			return nil
		}

		refreshExp := jwthelper.GetExpiration(n.UeParam[constants.REFRESH_TOKEN].(string))
		if time.Now().Before(refreshExp) {
			return n.RefreshCore()
		}
	}
	return n.LoginCore()
}

// RefreshCore 执行刷新token流程
func (n *Nucleus) RefreshCore() error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("刷新令牌流程异常: %v\n", err)
		}
	}()

	// 检查登录状态
	if n.UeParam[constants.AES_KEY] == nil ||
		n.UeParam[constants.ACCESS_TOKEN] == nil ||
		n.UeParam[constants.REFRESH_TOKEN] == nil {
		return fmt.Errorf("未登录，无法刷新令牌")
	}

	// 准备刷新请求
	req := models.NewUEReqbaseV2()
	param := models.UeConfigParam{
		Aeskey:     n.UeParam[constants.AES_KEY].(string),
		Privatekey: n.UeParam[constants.RSA_PRIVATEKEY].(string),
		Header: map[string]string{
			constants.ACCESS_TOKEN:  n.UeParam[constants.ACCESS_TOKEN].(string),
			constants.REFRESH_TOKEN: n.UeParam[constants.REFRESH_TOKEN].(string),
		},
	}

	// 重试配置
	maxRetries := 8
	baseDelay := time.Second
	var lastErr error

	// 执行刷新请求，带重试逻辑
	for i := 0; i < maxRetries; i++ {
		sdlog.Infof("开始第 %d 次刷新token尝试", i+1)

		resp, err := ue_api_v2.RefreshToken(req, param)
		if err == nil {
			if resp.Result <= 0 {
				if resp.Result == -9 || resp.Result == -8 || resp.Result == -7 {
					clear(n.UeParam) // 清空全局配置
				}
				sdlog.Errorf("刷新token失败，错误码: %d, 错误信息: %s", resp.Result, resp.Message)
				return fmt.Errorf("刷新token失败：%s", resp.Message)
			}

			// 更新配置信息
			n.UeParam[constants.ACCESS_TOKEN] = resp.Data.Accesstoken
			n.UeParam[constants.REFRESH_TOKEN] = resp.Data.Refreshtoken
			decryptedAesKey, decryptErr := rsa.Decrypt(resp.Data.Aeskey, n.UeParam[constants.RSA_PRIVATEKEY].(string))
			if decryptErr != nil {
				sdlog.Errorf("解密AES密钥失败: %v", decryptErr)
				return fmt.Errorf("解密AES密钥失败: %v", decryptErr)
			}
			n.UeParam[constants.AES_KEY] = decryptedAesKey
			if resp.Data.Serverpublickey != "" {
				n.UeParam[constants.SERVER_PUBLICKEY] = rsa.Base64ToPEM(resp.Data.Serverpublickey, true)
			}

			// 更新 UeConfigParam
			n.UeConfigParam = models.UeConfigParam2{
				Aeskey:      n.UeParam[constants.AES_KEY].(string),
				Privatekey:  n.UeParam[constants.RSA_PRIVATEKEY].(string),
				Accesstoken: n.UeParam[constants.ACCESS_TOKEN].(string),
			}

			sdlog.Infof("刷新token成功，配置信息:")
			// jsonStr, _ := json.MarshalIndent(n.UeParam, "", "  ")
			// sdlog.Infof(string(jsonStr))

			return nil
		}

		lastErr = err
		if i < maxRetries-1 {
			// 计算下一次重试的等待时间（指数退避）
			delay := baseDelay * time.Duration(1<<uint(i))
			sdlog.Errorf("刷新token失败，%d秒后重试第%d次: %v", delay/time.Second, i+2, err)
			time.Sleep(delay)
		}
	}

	sdlog.Errorf("刷新token失败，已重试%d次，最后一次错误: %v", maxRetries, lastErr)
	return fmt.Errorf("刷新token失败，已重试%d次，最后一次错误: %v", maxRetries, lastErr)
}
