package db

import (
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

const (
	UeSecretStatusActive = "active"
	UeSecretStatusClosed = "closed"
)

// GetUserUeSecret 查询用户当前有效的UE免密绑定
func GetUserUeSecret(tx *gorm.DB, userID int64) (*model.UserUeSecret, error) {
	var record model.UserUeSecret
	err := tx.Model(&model.UserUeSecret{}).
		Where("user_id = ? AND status = ?", userID, UeSecretStatusActive).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// BindUserUeSecret 绑定/更新用户UE免密支付
func BindUserUeSecret(tx *gorm.DB, userID int64, opennodecode, uenodecode string) error {
	// 校验 uenodecode 是否已被其他用户绑定
	var conflict model.UserUeSecret
	err := tx.Model(&model.UserUeSecret{}).
		Where("uenodecode = ? AND status = ? AND user_id != ?", uenodecode, UeSecretStatusActive, userID).
		First(&conflict).Error
	if err == nil {
		return gorm.ErrDuplicatedKey
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	var existing model.UserUeSecret
	err = tx.Model(&model.UserUeSecret{}).
		Where("user_id = ? AND status = ?", userID, UeSecretStatusActive).
		First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 新建绑定
			record := model.UserUeSecret{
				UserID:       userID,
				Opennodecode: opennodecode,
				Uenodecode:   uenodecode,
				Status:       UeSecretStatusActive,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			return tx.Create(&record).Error
		}
		return err
	}

	// 更新已有的绑定（例如换了节点）
	return tx.Model(&existing).Updates(map[string]interface{}{
		"opennodecode": opennodecode,
		"uenodecode":   uenodecode,
		"updated_at":   time.Now(),
	}).Error
}

// UnbindUserUeSecret 解绑用户UE免密支付
func UnbindUserUeSecret(tx *gorm.DB, userID int64) error {
	return tx.Model(&model.UserUeSecret{}).
		Where("user_id = ? AND status = ?", userID, UeSecretStatusActive).
		Updates(map[string]interface{}{
			"status":     UeSecretStatusClosed,
			"updated_at": time.Now(),
		}).Error
}
