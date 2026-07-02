package db

import (
	"TMA/service/db/model"

	"gorm.io/gorm"
)

func UserInfoById(tx *gorm.DB, uid int64) (*model.User, error) {
	var user model.User
	err := tx.Model(&model.User{}).Where("user_id = ?", uid).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UserLevelInfoById(tx *gorm.DB, level int32) (*model.UserLevelConfig, error) {
	var user model.UserLevelConfig
	err := tx.Model(&model.UserLevelConfig{}).Where("level_id=?", level).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
