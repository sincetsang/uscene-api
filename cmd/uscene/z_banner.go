package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

func banner(ec *middleware.AppRequestContext) error {
	now := time.Now()
	var list []model.Banner
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Banner{}).Where("show_start_at < ? and show_end_at > ?", now, now).Order("id desc").Find(&list).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(nil).Render(ec)
		}
		sdlog.Errorf("[banner] 查询banner失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(list).Render(ec)
}
