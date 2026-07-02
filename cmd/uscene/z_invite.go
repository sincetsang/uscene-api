package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service/db"
	"TMA/service/db/model"
)

type UserInviteRes struct {
	InviteCode string  `json:"invite_code"`
	InviteNum  int64   `json:"invite_num"`
	RewardSum  float64 `json:"reward_sum"`
}

// userInvite
func userInvite(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	uInfo, err := db.UserInfoById(ec.Nu.DB.WithContext(ec.Request().Context()), userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var inviteNum int64
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserInviteRecord{}).
		Where("inviter_id = ?", userId).
		Count(&inviteNum).Error

	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var rewardSum float64
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Where("user_id = ? and invitee_id != 0", userId).
		Select("COALESCE(SUM(rewards), 0)").
		Scan(&rewardSum).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(UserInviteRes{
		InviteCode: uInfo.InviteCode,
		InviteNum:  inviteNum,
		RewardSum:  rewardSum,
	}).Render(ec)
}
