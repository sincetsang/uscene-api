package db

import (
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

type CheckInPointWithOwnerV4 struct {
	model.CheckInPoint
	OwnerUsername string `json:"owner_username" gorm:"column:owner_username"`
	OwnerAvatar   string `json:"owner_avatar" gorm:"column:owner_avatar"`
	Liked         bool   `json:"liked" gorm:"column:liked"`
}

func GetAllValidCheckPointsByLaLoV4(tx *gorm.DB, latMin, latMax, lonMin, lonMax float64, OwnerId int64, keyword string, user_id int64, category string, hasAdvertisement int64, page, page_size int64) ([]*CheckInPointWithOwnerV4, int64, error) {
	var results []*CheckInPointWithOwnerV4
	var total int64

	// 构建基础查询条件（用于Count）
	baseQuery := tx.Model(&model.CheckInPoint{}).
		Where("check_in_point.latitude >= ? AND check_in_point.latitude <= ? AND check_in_point.longitude >= ? AND check_in_point.longitude <= ? AND check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1",
			latMin, latMax, lonMin, lonMax)

	// 添加条件查询
	if OwnerId != 0 {
		baseQuery = baseQuery.Where("owner_id = ?", OwnerId)
	}
	if keyword != "" {
		baseQuery = baseQuery.Where("title LIKE ?", "%"+keyword+"%")
	}
	if category != "" {
		baseQuery = baseQuery.Where("category = ?", category)
	}
	if hasAdvertisement != 0 {
		baseQuery = baseQuery.
			Joins("LEFT JOIN check_in_point_advertisement ON check_in_point.id = check_in_point_advertisement.check_in_point_id AND check_in_point_advertisement.status = 1 AND check_in_point_advertisement.start_time <= ? AND check_in_point_advertisement.end_time >= ?", time.Now(), time.Now())
	}

	// 查询总记录数
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 如果总记录数为0，直接返回空结果
	if total == 0 {
		return []*CheckInPointWithOwnerV4{}, 0, nil
	}

	// 构建完整查询（包含所有JOIN和字段）
	query := baseQuery.
		Select(`
			DISTINCT check_in_point.*,
			user.nickname as owner_username,
			user.avatar_url as owner_avatar,
			CASE WHEN user_like.id IS NOT NULL THEN 1 ELSE 0 END as liked
		`).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Joins("LEFT JOIN user_like ON check_in_point.id = user_like.spot_id AND user_like.user_id = ?", user_id)

	// 添加排序
	query = query.Order("check_in_point.created_at DESC")

	// 添加分页
	if page < 1 {
		page = 1
	}
	if page_size < 1 {
		page_size = 10 // 设置默认每页大小
	}
	offset := (page - 1) * page_size
	query = query.Offset(int(offset)).Limit(int(page_size))

	// 执行查询
	err = query.Find(&results).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []*CheckInPointWithOwnerV4{}, 0, nil
		}
		return nil, 0, err
	}

	return results, total, nil
}
