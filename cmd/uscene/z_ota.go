package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

// 获取最新升级信息
func otaLatest(ec *middleware.AppRequestContext) error {
	// 获取平台类型
	platform := ec.Request().Header.Get("Platform")
	if platform == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询最新的升级信息
	var ota model.Otum
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("platform = ? AND is_active = ?", platform, true).
		Order("created_at DESC").
		First(&ota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(nil).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建返回数据
	type OtaResponse struct {
		Version      string    `json:"version"`       // 版本号
		Platform     string    `json:"platform"`      // 平台类型
		ForceUpdate  string    `json:"force_update"`  // 是否强制更新
		DownloadURL  string    `json:"download_url"`  // 下载地址
		ReleaseNotes string    `json:"release_notes"` // 更新说明
		MinVersion   string    `json:"min_version"`   // 最低支持版本
		FileSize     int64     `json:"file_size"`     // 文件大小
		Md5          string    `json:"md5"`           // 文件MD5
		CreatedAt    time.Time `json:"created_at"`    // 创建时间
		NeedUpdate   bool      `json:"need_update"`   // 是否需要升级
	}

	response := OtaResponse{
		Version:      ota.Version,
		Platform:     ota.Platform,
		ForceUpdate:  ota.ForceUpdate,
		DownloadURL:  ota.DownloadURL,
		ReleaseNotes: ota.ReleaseNotes,
		MinVersion:   ota.MinVersion,
		FileSize:     ota.FileSize,
		Md5:          ota.Md5,
		CreatedAt:    ota.CreatedAt,
		NeedUpdate:   service.NeedUpdate(ec.Request().Header.Get("App-Version"), ota.Version),
	}

	return webapi.OK(response).Render(ec)
}
