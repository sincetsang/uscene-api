package db

import (
	"TMA/service/db/model"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func UpdateUserMemberByMemberOrder(tx *gorm.DB, orderId int64) error {
	var order model.MemberOrder
	err := tx.Model(&model.MemberOrder{}).Where("order_id = ?", orderId).First(&order).Error
	if err != nil {
		return err
	}

	periodInt, err := strconv.ParseInt(order.Period, 10, 64)
	if err != nil {
		return err
	}
	// 查询会员信息，如果没有则新创建
	var member model.SupermapCheckinmember
	err = tx.Model(&model.SupermapCheckinmember{}).Where("user_id = ?", order.UserID).First(&member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			member = model.SupermapCheckinmember{
				UserID:           order.UserID,
				ExpiredTimestamp: int32(time.Now().Add(time.Duration(periodInt) * 24 * time.Hour).Unix()),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			err = tx.Model(&model.SupermapCheckinmember{}).Create(&member).Error
			if err != nil {
				return err
			}
			return err
		}
		return err
	}

	// 有会员信息，则先比较过期时间，如果过期时间小于当前时间，则新的过期时间=当前时间+会员周期
	// 如果过期时间大于当前时间，则在当前过期时间基础上加上会员周期
	if member.ExpiredTimestamp < int32(time.Now().Unix()) {
		member.ExpiredTimestamp = int32(time.Now().Add(time.Duration(periodInt) * 24 * time.Hour).Unix())
		err = tx.Model(&model.SupermapCheckinmember{}).Where("user_id = ?", order.UserID).Update("expired_timestamp", member.ExpiredTimestamp).Error
		if err != nil {
			return err
		}
	}
	if member.ExpiredTimestamp >= int32(time.Now().Unix()) {
		member.ExpiredTimestamp = int32(member.ExpiredTimestamp + int32(periodInt*86400))
		err = tx.Model(&model.SupermapCheckinmember{}).Where("user_id = ?", order.UserID).Update("expired_timestamp", member.ExpiredTimestamp).Error
		if err != nil {
			return err
		}
	}
	return nil
}
