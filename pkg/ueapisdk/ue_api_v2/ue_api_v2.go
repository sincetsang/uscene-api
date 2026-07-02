package ue_api_v2

// 修复导入部分
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/utils"
	"github.com/sulink/ueapisdk/utils/aes"
	"github.com/sulink/ueapisdk/utils/constants" // 确保常量包存在
	"github.com/sulink/ueapisdk/utils/rsa"
)

var (
	baseURL string
	client  = &http.Client{Timeout: 30 * time.Second}
)

func Initialization(baseUrl string) {
	if len(baseUrl) > 0 && baseUrl[len(baseUrl)-1] != '/' {
		baseUrl += "/"
	}
	baseURL = baseUrl
}

func GetConfig(req models.ConfigReq) (models.UERespbaseV2[models.ConfigDto], error) {
	url := baseURL + "svr/AuthSvr/GetConfig"
	return post[models.ConfigReq, models.UERespbaseV2[models.ConfigDto]](url, req, models.UeConfigParam{})
}

func Login(req models.ApiLoginReq, config models.UeConfigParam) (models.UERespbaseV2[models.LoginDto], error) {
	url := baseURL + "svr/AuthSvr/Login"
	return post[models.ApiLoginReq, models.UERespbaseV2[models.LoginDto]](url, req, config)
}

func RefreshToken(req models.UEReqbaseV2, config models.UeConfigParam) (models.UERespbaseV2[models.LoginDto], error) {
	url := baseURL + "svr/AuthSvr/RefreshToken"

	// 记录请求信息
	reqJson, _ := json.Marshal(req)

	// 构建curl命令
	curlCmd := fmt.Sprintf("curl -X POST '%s' \\\n", url)
	curlCmd += fmt.Sprintf("  -H 'Content-Type: application/json' \\\n")

	// 添加配置中的header
	if config.Header != nil {
		for k, v := range config.Header {
			curlCmd += fmt.Sprintf("  -H '%s: %s' \\\n", k, v)
		}
	}

	// 添加AES加密相关的header
	if config.Aeskey != "" && config.Privatekey != "" {
		aesIV := utils.GetRandomString(16)
		encrypted, _ := aes.EncryptToBase64(string(reqJson), config.Aeskey, aesIV)
		signature, _ := rsa.Sign(encrypted, config.Privatekey)

		curlCmd += fmt.Sprintf("  -H '%s: %s' \\\n", constants.AES_IV, aesIV)
		curlCmd += fmt.Sprintf("  -H '%s: %s' \\\n", constants.SIGN_BODY, signature)
		curlCmd += fmt.Sprintf("  -d '%s'", encrypted)
	} else {
		curlCmd += fmt.Sprintf("  -d '%s'", string(reqJson))
	}

	// 使用logrus包记录日志（带时间戳和更多上下文信息）
	logrus.WithFields(logrus.Fields{
		"method": "RefreshToken",
		"url":    url,
	}).Infof("Curl Command:\n%s", curlCmd)

	return post[models.UEReqbaseV2, models.UERespbaseV2[models.LoginDto]](url, req, config)
}

func PayCallValidate(req models.PayValidateReq, config models.UeConfigParam2) (models.UERespbase, error) {
	url := baseURL + "svr/PurseSvr/PayCallValidate"
	return post[models.PayValidateReq, models.UERespbase](
		url,
		req,
		models.UeConfigParam{
			Aeskey:     config.Aeskey,
			Privatekey: config.Privatekey,
			Header:     map[string]string{"accesstoken": config.Accesstoken},
		},
	)
}

func PayCallValidateV2(req models.PayValidateReqV2, config models.UeConfigParam2) (models.UERespbase, error) {
	url := baseURL + "svr/PurseSvr/PayCallValidateV2"
	return post[models.PayValidateReqV2, models.UERespbase](
		url,
		req,
		models.UeConfigParam{
			Aeskey:     config.Aeskey,
			Privatekey: config.Privatekey,
			Header:     map[string]string{"accesstoken": config.Accesstoken},
		},
	)
}

func CreateOrder(req models.CreateOrderReq, config models.UeConfigParam2) (models.UERespbase, error) {
	url := baseURL + "svr/PurseSvr/CreateOrder"
	return post[models.CreateOrderReq, models.UERespbase](
		url,
		req,
		models.UeConfigParam{
			Aeskey:     config.Aeskey,
			Privatekey: config.Privatekey,
			Header:     map[string]string{"accesstoken": config.Accesstoken},
		},
	)
}

func CreateOrderV2(req models.CreateOrderV2Req, config models.UeConfigParam2) (models.UERespbase, error) {
	url := baseURL + "svr/PurseSvr/CreateOrderV2"
	return post[models.CreateOrderV2Req, models.UERespbase](
		url,
		req,
		models.UeConfigParam{
			Aeskey:     config.Aeskey,
			Privatekey: config.Privatekey,
			Header:     map[string]string{"accesstoken": config.Accesstoken},
		},
	)
}

func UpdateTCodeStatus(req models.TCodeReq, config models.UeConfigParam2) (models.UERespbase, error) {
	url := baseURL + "svr/SicIndexSvr/UpdateTCodeStatus"
	return post[models.TCodeReq, models.UERespbase](
		url,
		req,
		models.UeConfigParam{
			Aeskey:     config.Aeskey,
			Privatekey: config.Privatekey,
			Header:     map[string]string{"accesstoken": config.Accesstoken},
		},
	)
}

func GetResponse[T any](body string, signBody string, publicKey string, aesKey string, aesIV string) (*T, error) {
	// 参数校验
	if aesIV == "" || signBody == "" {
		return nil, fmt.Errorf("header参数aesiv,signbody不存在")
	}

	// 签名验证
	if valid := rsa.CheckSign(body, signBody, publicKey); !valid {
		return nil, fmt.Errorf("签名验证失败")
	}

	// AES解密
	decrypted, err := aes.DecryptFromBase64(body, aesKey, aesIV)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	// 反序列化
	// 在解密后添加时间格式预处理
	decrypted = replaceDateTimeFormat(decrypted)
	var result T
	if err := json.Unmarshal([]byte(decrypted), &result); err != nil {
		return nil, fmt.Errorf("反序列化失败: %v", err)
	}

	return &result, nil
}

func post[TReq any, TResp any](url string, req TReq, config models.UeConfigParam) (TResp, error) {
	reqBody, _ := json.Marshal(req)
	var zero TResp

	// 修复点1：添加请求头处理逻辑
	headers := make(map[string]string)
	if config.Header != nil {
		headers = config.Header
	}

	// 修复点2：正确处理加密和非加密两种请求路径
	if config.Aeskey != "" && config.Privatekey != "" {
		aesIV := utils.GetRandomString(16)
		encrypted, err := aes.EncryptToBase64(string(reqBody), config.Aeskey, aesIV)
		if err != nil {
			return zero, err
		}

		signature, err := rsa.Sign(encrypted, config.Privatekey)
		if err != nil {
			return zero, err
		}

		// 修复点3：保持与Java一致的header设置逻辑
		headers[constants.AES_IV] = aesIV
		headers[constants.SIGN_BODY] = signature
		reqBody = []byte(encrypted)
	}

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// 统一设置headers（包含可能存在的自定义headers）
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// 修复点4：响应解密逻辑与Java保持一致
	if aesIV := resp.Header.Get(constants.AES_IV); aesIV != "" {
		decrypted, err := aes.DecryptFromBase64(string(bodyBytes), config.Aeskey, aesIV)
		if err != nil {
			return zero, err
		}
		bodyBytes = []byte(decrypted)
	}

	return unmarshalResponse[TResp](bodyBytes)
}

// 修复辅助函数
func unmarshalResponse[T any](body []byte) (T, error) {
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}
	return result, nil
}

func replaceDateTimeFormat(jsonStr string) string {
	// 正则匹配 "yyyy-MM-dd HH:mm:ss" 格式
	re := regexp.MustCompile(`"(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2})"`)
	return re.ReplaceAllString(jsonStr, `"${1}T${2}+08:00"`) // 添加时区信息
}
