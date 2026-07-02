package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"strconv"
)

func userLevelInfo(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	info, err := service.UserLevelInfo(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		sdlog.Errorf("[userLevelInfo] 查询用户等级信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

func userUpgrade(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	uidStr := strconv.FormatInt(userId, 10)
	ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, uidStr)
	if !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}
	err := service.UserUpgrade(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		sdlog.Errorf("[userUpgrade] 用户升级失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}
