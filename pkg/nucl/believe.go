package nucl

import (
	"TMA/pkg/sdlog"
	"encoding/json"
	"fmt"
	"net/http"
)

type BelieveClient struct {
	Client *http.Client
	Domain string
	Secret string
}

// GetChatLinkResponse 获取聊天链接的响应结构
type GetChatLinkResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

func (nu *Nucleus) initBelieve() error {
	if !nu.Options.Believe {
		return nil
	}
	return nu.withLog("Believe", func() error {
		// 初始化 client 配置
		client := &BelieveClient{
			Client: &http.Client{},
			Domain: nu.Config.Believe.Domain,
			Secret: nu.Config.Believe.Secret,
		}
		nu.BelieveClient = client
		return nil
	})
}

// GetChatLink 获取聊天链接
func (c *BelieveClient) GetChatLink(believeID string) (string, error) {
	url := fmt.Sprintf("%s/user/data/getChatLink/%s?secret=%s", c.Domain, believeID, c.Secret)

	resp, err := c.Client.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取聊天链接失败: %v", err)
	}
	defer resp.Body.Close()

	var result GetChatLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		sdlog.Error("解析响应失败: %v,响应结构为: %v", err, result)
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("获取聊天链接失败: %s", result.Msg)
	}

	return result.Data, nil
}
