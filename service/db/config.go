package db

import (
	"TMA/service/db/model"
	"gorm.io/gorm"
)

func GetConfigByKey(tx *gorm.DB, _key string, defaultValue string) *string {
	config := model.Config{}
	err := tx.Model(&model.Config{}).Where("_key = ?", _key).First(&config).Error
	if err != nil {
		return &defaultValue
	}
	return &config.Value
}

func GetConfigs(tx *gorm.DB) ([]model.Config, error) {
	configs := []model.Config{}
	err := tx.Model(&model.Config{}).Find(&configs).Error
	if err != nil {
		return configs, err
	}
	return configs, nil
}
