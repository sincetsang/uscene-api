package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"strconv"
	"time"
)

// 获取对话列表
func getConversations(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	conversations, err := service.GetConversations(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(conversations).Render(ec)
}

// 获取对话详情
func getConversation(ec *middleware.AppRequestContext) error {
	conversationId := ec.Param("id")
	userId := ec.AuthData.User.ID

	messages, err := service.GetConversation(ec.Request().Context(), ec.Nu, conversationId, userId)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	return webapi.OK(messages).Render(ec)
}

// 发送消息
func sendMessage(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 解析请求参数
	var req struct {
		ConversationID string `json:"conversation_id"`
		Content        string `json:"content" binding:"required"`
		IsNewChat      bool   `json:"is_new_chat"` // 是否为新对话
		City           string `json:"city"`        // 城市
	}

	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 加锁限制消息数量
	lockStr := "chat:" + strconv.FormatInt(userId, 10)
	ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockStr)
	if !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}
	// 查询用户今日消息数
	var messageCount int64
	today := time.Now().Format("2006-01-02")
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("chat_message").
		Where("user_id = ? AND DATE(created_at) = ?", userId, today).
		Count(&messageCount).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 如果今日消息数超过10条,限制发送
	if messageCount >= 10 {
		return webapi.Error(common.ErrTooManyRequests).Render(ec)
	}

	// 如果是新对话，不需要ConversationID
	if !req.IsNewChat && req.ConversationID == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	aiMessage, err := service.SendMessage(
		ec.Request().Context(),
		ec.Nu,
		userId,
		req.ConversationID,
		req.Content,
		req.IsNewChat,
		req.City,
	)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(aiMessage).Render(ec)
}

// 删除对话
func deleteConversation(ec *middleware.AppRequestContext) error {
	conversationId := ec.Param("id")
	userId := ec.AuthData.User.ID

	err := service.DeleteConversation(ec.Request().Context(), ec.Nu, conversationId, userId)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	return webapi.OK(nil).Render(ec)
}

// 获取推荐对话列表
func getRecommendedConversations(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	configs, err := service.GetRecommendedConversations(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(configs).Render(ec)
}
