package nucl

import (
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

type EmailConfig struct {
	AccessKeyID     string `toml:"access_key_id"`
	AccessKeySecret string `toml:"access_key_secret"`
	AccountName     string `toml:"account_name"`
}

func (nu *Nucleus) initEmail() error {
	if !nu.Options.Email {
		return nil
	}
	return nu.withLog("Email", func() error {
		// 初始化 client 配置
		config := &openapi.Config{
			AccessKeyId:     tea.String(nu.Config.Email.AccessKeyID),
			AccessKeySecret: tea.String(nu.Config.Email.AccessKeySecret),
		}
		config.Endpoint = tea.String("dm.aliyuncs.com")

		// 创建 client
		client, err := dm.NewClient(config)
		if err != nil {
			return err
		}
		nu.EmailClient = client
		return nil
	})
}

// SendEmail 发送邮件
func (nu *Nucleus) SendEmail(toAddress, code string) error {
	if nu.EmailClient == nil {
		return fmt.Errorf("email client not initialized")
	}

	htmlBody := "<p>验证码为：" + code + "</p>"
	// 构造发送请求
	sendReq := &dm.SingleSendMailRequest{
		AccountName:    tea.String(nu.Config.Email.AccountName),
		AddressType:    tea.Int32(1), // 1 表示使用发信地址为发件人地址
		ToAddress:      tea.String(toAddress),
		Subject:        tea.String("优享派验证码"),
		HtmlBody:       tea.String(htmlBody),
		ReplyToAddress: tea.Bool(true),
	}

	// 发送邮件
	_, err := nu.EmailClient.SingleSendMail(sendReq)
	return err
}
