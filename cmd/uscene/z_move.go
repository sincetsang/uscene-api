package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db"
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

type MoveRes struct {
	Rewards int64 `json:"rewards"`
}

func move(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := []service.LocationObject{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	moveAt := service.CalculateMoveTime(req)
	if moveAt <= 0 {
		return webapi.OK(MoveRes{Rewards: 0}).Render(ec)
	}
	uinfo, err := db.UserInfoById(ec.Nu.DB.WithContext(ec.Request().Context()), userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 获取当前时间并比较 MovedAt 是否为今天
	now := time.Now()
	if !isToday(uinfo.MovedAt, now) {
		// 如果不是今天，重置 RemainingStamina 为 Stamina
		uinfo.RemainingStamina = uinfo.Stamina
	}
	if uinfo.RemainingStamina == 0 {
		return webapi.OK(MoveRes{Rewards: 0}).Render(ec)
	}
	if moveAt > int(uinfo.RemainingStamina) {
		moveAt = int(uinfo.RemainingStamina)
	}

	uLevelInfo, err := db.UserLevelInfoById(ec.Nu.DB.WithContext(ec.Request().Context()), uinfo.Level)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 根据 moveAt 计算奖励
	rewards := int64(moveAt) * uLevelInfo.StaminaReward
	// 更新用户余额和体力值
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Transaction(func(tx *gorm.DB) error {
		// 更新用户余额和体力值
		if err = tx.Model(&model.User{}).
			Where("user_id = ?", userId).
			Updates(map[string]interface{}{
				"balance":           uinfo.Balance + float64(rewards),
				"remaining_stamina": uinfo.RemainingStamina - int32(moveAt),
				"moved_at":          now,
			}).Error; err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
		// 更新用户资产明细
		assetDetail := model.UserAssetDetail{
			UserID:   userId,
			Category: common.ConfigAssetTypeByMove,
			Currency: common.ConfigPlatformCoinName,
			Rewards:  float64(rewards),
		}
		if err = tx.Model(&model.UserAssetDetail{}).Create(&assetDetail).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(MoveRes{
		Rewards: rewards,
	}).Render(ec)

}

// 判断日期是否为今天
func isToday(date, current time.Time) bool {
	return date.Year() == current.Year() && date.YearDay() == current.YearDay()
}
