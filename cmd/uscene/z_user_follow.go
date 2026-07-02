package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

type UserFollowParam struct {
	UserId int64 `json:"user_id"`
}

type UserFollowResponse struct {
	IsFollowed bool `json:"is_followed"`
}

func userFollow(ec *middleware.AppRequestContext) error {

	req := UserFollowParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.UserId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.UserId == ec.AuthData.User.ID {
		return webapi.Error(common.ErrUserFollowYourself).Render(ec)
	}

	// 检查user是否存在
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("user_id = ?", req.UserId).First(&model.User{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrUserNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	var isFollowed bool
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapUserfollow{}).Where("user_id = ? and followed_user_id = ?", ec.AuthData.User.ID, req.UserId).First(&model.SupermapUserfollow{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建关注记录
			err = ec.Nu.DB.WithContext(ec.Request().Context()).Create(&model.SupermapUserfollow{
				UserID:         ec.AuthData.User.ID,
				FollowedUserID: req.UserId,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}).Error
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
			isFollowed = true
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	} else {
		// 如果存在则删除关注记录
		err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapUserfollow{}).Where("user_id = ? and followed_user_id = ?", ec.AuthData.User.ID, req.UserId).Delete(&model.SupermapUserfollow{}).Error
		if err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
		isFollowed = false
	}

	var changeValue int64
	if isFollowed {
		changeValue = 1
	} else {
		changeValue = -1
	}
	// 如果是follow ，用户的关注人数+1，被关注用户粉丝数+1
	// 如果是unfollow ，用户的关注人数-1，被关注用户粉丝数-1

	err = service.CreateUserStat(ec.Request().Context(), ec.Nu, ec.AuthData.User.ID)
	if err != nil {
		sdlog.Errorf("[userFollow] 创建用户统计数据失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	err = service.CreateUserStat(ec.Request().Context(), ec.Nu, req.UserId)
	if err != nil {
		sdlog.Errorf("[userFollow] 创建用户统计数据失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 更新用户统计数据
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapUserstat{}).Where("followers_count + ? >= 0", changeValue).Where("user_id = ?", req.UserId).Update("followers_count", gorm.Expr("followers_count + ?", changeValue)).Error
	if err != nil {
		sdlog.Errorf("[userFollow] 更新用户统计数据失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapUserstat{}).Where("following_count + ? >= 0", changeValue).Where("user_id = ?", ec.AuthData.User.ID).Update("following_count", gorm.Expr("following_count + ?", changeValue)).Error
	if err != nil {
		sdlog.Errorf("[userFollow] 更新用户统计数据失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(UserFollowResponse{
		IsFollowed: isFollowed,
	}).Render(ec)

}
