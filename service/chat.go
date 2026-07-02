package service

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/service/db/model"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type chatOpenaiRequest struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	Stream         bool      `json:"stream"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type chatOpenaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Images  []struct {
				URL string `json:"url"`
			} `json:"images,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// 调用AI服务获取回复
func getAIResponse(apiKey string, messages []message) (string, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		reqBody, err := json.Marshal(chatOpenaiRequest{
			// Model: "gpt-4o",
			Model:    "gpt-3.5-turbo",
			Messages: messages,
			Stream:   false,
			ResponseFormat: struct {
				Type string `json:"type"`
			}{
				Type: "text",
			},
		})
		if err != nil {
			return "", fmt.Errorf("marshal request failed: %v", err)
		}

		client := &http.Client{Timeout: requestTimeout}
		req, err := http.NewRequest("POST", openaiAPI, bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("create request failed: %v", err)
			sdlog.Warnf("AI服务调用第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("AI request failed: %v", err)
			sdlog.Warnf("AI服务调用第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("AI request failed with status %d: %s", resp.StatusCode, string(body))
			sdlog.Warnf("AI服务调用第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		var result chatOpenaiResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			lastErr = fmt.Errorf("decode response failed: %v", err)
			sdlog.Warnf("AI服务调用第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		if len(result.Choices) == 0 {
			lastErr = fmt.Errorf("no AI response")
			sdlog.Warnf("AI服务调用第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		// 处理文本和图片内容
		content := result.Choices[0].Message.Content
		if len(result.Choices[0].Message.Images) > 0 {
			content += "\n\n图片：\n"
			for _, img := range result.Choices[0].Message.Images {
				content += img.URL + "\n"
			}
		}

		return content, nil
	}

	return "", fmt.Errorf("after %d retries: %v", maxRetries, lastErr)
}

// 获取对话列表
func GetConversations(ctx context.Context, nu *nucl.Nucleus, userId int64) ([]model.ChatConversation, error) {
	var conversations []model.ChatConversation
	err := nu.DB.WithContext(ctx).
		Model(&model.ChatConversation{}).
		Where("user_id = ? AND is_active = ?", userId, true).
		Order("updated_at DESC").
		Find(&conversations).Error

	if err != nil {
		return nil, err
	}

	return conversations, nil
}

// 获取对话详情
func GetConversation(ctx context.Context, nu *nucl.Nucleus, conversationId string, userId int64) ([]model.ChatMessage, error) {
	var conversation model.ChatConversation
	err := nu.DB.WithContext(ctx).
		Model(&model.ChatConversation{}).
		Where("id = ? AND user_id = ?", conversationId, userId).
		First(&conversation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrParam
		}
		sdlog.Errorf("[GetConversation] 获取对话详情失败: %v", err)
		return nil, common.ErrService
	}

	var messages []model.ChatMessage
	err = nu.DB.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("conversation_id = ?", conversationId).
		Order("created_at ASC").
		Find(&messages).Error

	if err != nil {
		sdlog.Errorf("[GetConversation] 获取对话消息失败: %v", err)
		return nil, common.ErrService
	}

	return messages, nil
}

// 发送消息
func SendMessage(ctx context.Context, nu *nucl.Nucleus, userId int64, conversationId string, content string, isNewChat bool, city string) (*model.ChatMessage, error) {
	var conversation model.ChatConversation

	// 如果是新对话，创建对话
	if isNewChat {
		// 设置标题，如果内容超过20字符则截断并添加省略号
		title := content
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20]) + "..."
		}

		conversation = model.ChatConversation{
			ID:        sdrand.GenerateSecureString(13, true, false, true),
			Title:     title,
			UserID:    userId,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := nu.DB.WithContext(ctx).Create(&conversation).Error; err != nil {
			sdlog.Errorf("[SendMessage] 创建对话失败: %v", err)
			return nil, common.ErrService
		}
		conversationId = conversation.ID
	} else {
		// 检查对话是否存在
		if conversationId == "" {
			sdlog.Errorf("[SendMessage] 对话ID不能为空")
			return nil, common.ErrParam
		}

		err := nu.DB.WithContext(ctx).
			Model(&model.ChatConversation{}).
			Where("id = ? AND user_id = ?", conversationId, userId).
			First(&conversation).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				sdlog.Errorf("[SendMessage] 对话不存在: %v", err)
				return nil, common.ErrParam
			}
			sdlog.Errorf("[SendMessage] 获取对话失败: %v", err)
			return nil, common.ErrService
		}
	}

	// 创建用户消息
	userMessage := model.ChatMessage{
		Role:           "user",
		UserID:         userId,
		Content:        content,
		ConversationID: conversationId,
		CreatedAt:      time.Now(),
		IsError:        false,
	}

	if err := nu.DB.WithContext(ctx).Create(&userMessage).Error; err != nil {
		sdlog.Errorf("[SendMessage] 创建用户消息失败: %v", err)
		return nil, common.ErrService
	}

	// 获取历史消息
	var historyMessages []model.ChatMessage
	var err error
	err = nu.DB.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("conversation_id = ?", conversationId).
		Order("created_at desc").Limit(10).
		Find(&historyMessages).Error

	if err != nil {
		sdlog.Errorf("[SendMessage] 获取历史消息失败: %v", err)
		return nil, common.ErrService
	}

	// 反转消息顺序，使其按时间正序排列
	for i, j := 0, len(historyMessages)-1; i < j; i, j = i+1, j-1 {
		historyMessages[i], historyMessages[j] = historyMessages[j], historyMessages[i]
	}

	// 构建AI请求消息
	var messages []message
	// 添加system消息，设置AI为全球旅行专家
	messages = append(messages, message{
		Role: "system",
		Content: `你是一位经验丰富的全球旅行专家，精通世界各地的旅游景点、文化特色、美食佳肴、交通住宿等。你可以为用户提供专业的旅行建议、行程规划、预算控制、安全提示等全方位的旅行咨询服务。

在回答问题时，请尽可能提供丰富、详细的内容，包括但不限于：

1. 基本信息：
   - 景点/地点的历史背景
   - 文化特色和意义
   - 最佳游览时间
   - 门票价格和开放时间
   - 交通方式和路线
   - 周边配套设施

2. 实用建议：
   - 游览路线推荐
   - 拍照打卡点
   - 特色美食推荐
   - 购物指南
   - 注意事项
   - 省钱技巧
   - 安全提示

3. 深度体验：
   - 当地文化体验
   - 特色活动推荐
   - 小众景点介绍
   - 季节性活动
   - 节日庆典
   - 当地人的生活方式

4. 内容组织：
   - 使用清晰的标题和分段
   - 重要信息加粗或高亮
   - 使用列表和表格整理信息
   - 添加实用的小贴士
   - 提供相关链接和参考

请确保：
1. 内容详实但不冗长
2. 信息准确且实用
3. 语言生动有趣
4. 适当使用emoji增加趣味性
5. 描述要具体形象，帮助用户想象场景

记住：你的回答要像一位经验丰富的导游，既专业又亲切，让用户通过你的文字描述感受到身临其境的体验。
` + city + ` 作为用户当前所在城市，请基于以下原则推荐：
1. 已提供城市：根据该城市的地标、美食、文化特色等进行个性化推荐
2. 未指定城市：基于用户位置进行就近推荐
3. 直接给出推荐内容，无需询问或确认用户所在城市
4. 结合用户具体需求，提供有针对性的建议`,
	})
	// 添加历史消息
	for _, msg := range historyMessages {
		messages = append(messages, message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 调用AI服务获取回复
	aiResponse, err := getAIResponse(nu.OpenaiApiKey, messages)
	if err != nil {
		sdlog.Errorf("[SendMessage] 调用AI服务失败: %v", err)
		return nil, common.ErrService
	}

	// 创建AI回复消息
	aiMessage := model.ChatMessage{
		UserID:         userId,
		Role:           "assistant",
		Content:        aiResponse,
		ConversationID: conversationId,
		CreatedAt:      time.Now(),
		IsError:        false,
	}

	if err := nu.DB.WithContext(ctx).Create(&aiMessage).Error; err != nil {
		sdlog.Errorf("[SendMessage] 创建AI回复消息失败: %v", err)
		return nil, common.ErrService
	}

	// 更新对话的更新时间
	conversation.UpdatedAt = time.Now()
	if err := nu.DB.WithContext(ctx).Save(&conversation).Error; err != nil {
		sdlog.Errorf("[SendMessage] 更新对话失败: %v", err)
		return nil, common.ErrService
	}

	return &aiMessage, nil
}

// 删除对话
func DeleteConversation(ctx context.Context, nu *nucl.Nucleus, conversationId string, userId int64) error {
	var conversation model.ChatConversation
	err := nu.DB.WithContext(ctx).
		Model(&model.ChatConversation{}).
		Where("id = ? AND user_id = ?", conversationId, userId).
		First(&conversation).Error

	if err != nil {
		return err
	}

	// 软删除对话
	conversation.IsActive = false
	if err := nu.DB.WithContext(ctx).Save(&conversation).Error; err != nil {
		return err
	}

	return nil
}

// GetRecommendedConversations 获取推荐的对话列表
func GetRecommendedConversations(ctx context.Context, nu *nucl.Nucleus, userId int64) ([]model.ChatConfig, error) {
	var configs []model.ChatConfig
	if err := nu.DB.Where("is_active = ?", true).Order("sort_order asc").Limit(10).Find(&configs).Error; err != nil {
		return nil, sderr.WithStack(err)
	}
	return configs, nil
}
