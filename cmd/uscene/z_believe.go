package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
)

type BelieveChatLinkResponse struct {
	Link string `json:"link"`
}

func getBelieveChatLink(ec *middleware.AppRequestContext) error {
	believeID := ec.Request().URL.Query().Get("believe_id")
	if believeID == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 获取聊天链接
	link, err := ec.Nu.BelieveClient.GetChatLink(believeID)
	if err != nil {
		sdlog.Errorf("[getBelieveChatLink] 获取聊天链接失败: %v", err)
		return webapi.Error(common.ErrBelieveChatLinkNotFound).Render(ec)
	}

	return webapi.OK(BelieveChatLinkResponse{
		Link: link,
	}).Render(ec)
}
