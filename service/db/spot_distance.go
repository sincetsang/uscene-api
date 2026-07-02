package db

import (
	"TMA/service/db/model"

	"gorm.io/gorm"
)

type CheckInPointWithOwnerByDistance struct {
	model.CheckInPoint
	OwnerUsername string  `json:"owner_username" gorm:"column:owner_username"`
	OwnerAvatar   string  `json:"owner_avatar" gorm:"column:owner_avatar"`
	Distance      float64 `json:"distance" gorm:"column:distance"`
	CategoryImage string  `json:"category_image"` // 地标分类图片
}

func GetAllValidCheckPointsByLaLoByDistance(tx *gorm.DB, latitude, longitude float64, user_id int64, category string, page, page_size int64) ([]*CheckInPointWithOwnerByDistance, int64, error) {
	var results []*CheckInPointWithOwnerByDistance
	var total int64

	// 构建基础查询条件（用于Count）
	baseQuery := tx.Model(&model.CheckInPoint{}).
		Where("check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1")

	// 添加条件查询
	if category != "" {
		baseQuery = baseQuery.Where("category = ?", category)
	}

	// 查询总记录数
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 如果总记录数为0，直接返回空结果
	if total == 0 {
		return []*CheckInPointWithOwnerByDistance{}, 0, nil
	}

	// 构建完整查询（包含所有JOIN和字段）
	query := baseQuery.
		Select(`
        check_in_point.*,
        user.nickname as owner_username,
        user.avatar_url as owner_avatar,
        ST_Distance_Sphere(point(check_in_point.longitude, check_in_point.latitude), point(?, ?)) AS distance
    `, longitude, latitude).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id")
	// 添加排序
	query = query.Order("distance ASC")

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
			return []*CheckInPointWithOwnerByDistance{}, 0, nil
		}
		return nil, 0, err
	}

	return results, total, nil
}

func GetAllValidCheckPointsByIds(tx *gorm.DB, latitude, longitude float64, spot_ids []int64) ([]*CheckInPointWithOwnerByDistance, int64, error) {
	var results []*CheckInPointWithOwnerByDistance
	var total int64

	// 构建基础查询条件（用于Count）
	baseQuery := tx.Model(&model.CheckInPoint{}).
		Where("check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1")

	// 查询总记录数
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 如果总记录数为0，直接返回空结果
	if total == 0 {
		return []*CheckInPointWithOwnerByDistance{}, 0, nil
	}

	// 构建完整查询（包含所有JOIN和字段）
	query := baseQuery.
		Select(`
        check_in_point.*,
        user.nickname as owner_username,
        user.avatar_url as owner_avatar,
        ST_Distance_Sphere(point(check_in_point.longitude, check_in_point.latitude), point(?, ?)) AS distance
    `, longitude, latitude).Where("check_in_point.id IN ?", spot_ids).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id")
	// 添加排序
	query = query.Order("distance ASC")

	// 执行查询
	err = query.Find(&results).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []*CheckInPointWithOwnerByDistance{}, 0, nil
		}
		return nil, 0, err
	}

	return results, total, nil
}
