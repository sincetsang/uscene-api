package db

import (
	"TMA/service/db/model"
	"time"

	"gorm.io/gorm"
)

type CheckInPointWithOwnerV2 struct {
	model.CheckInPoint
	OwnerUsername string `json:"owner_username" gorm:"column:owner_username"`
	OwnerAvatar   string `json:"owner_avatar" gorm:"column:owner_avatar"`
	Liked         bool   `json:"liked" gorm:"column:liked"`
	TelegramUid   string `json:"telegram_uid" gorm:"column:telegram_uid"`
	WhatsappUid   string `json:"whatsapp_uid" gorm:"column:whatsapp_uid"`
	PcnUid        string `json:"pcn_uid" gorm:"column:pcn_uid"`
}

func GetAllValidCheckPointsByLaLoV2(tx *gorm.DB, latMin, latMax, lonMin, lonMax float64, OwnerId int64, keyword string, user_id int64, category string, hasAdvertisement int64) ([]*CheckInPointWithOwnerV2, error) {
	var results []*CheckInPointWithOwnerV2

	// 构建基础查询
	query := tx.Model(&model.CheckInPoint{}).
		Select(`
			DISTINCT check_in_point.*,
			user.nickname as owner_username,
			user.avatar_url as owner_avatar,
			CASE WHEN user_like.id IS NOT NULL THEN 1 ELSE 0 END as liked
		`).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Joins("LEFT JOIN user_like ON check_in_point.id = user_like.spot_id AND user_like.user_id = ?", user_id).
		//Where("check_in_point.latitude >= ? AND check_in_point.latitude <= ? AND check_in_point.longitude >= ? AND check_in_point.longitude <= ? AND check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1",
		Where("check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1")

	// 添加条件查询
	if OwnerId != 0 {
		query = query.Where("owner_id = ?", OwnerId)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%").Limit(30)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if hasAdvertisement != 0 {
		query = query.Where("check_in_point.id IN (SELECT check_in_point_id FROM check_in_point_advertisement WHERE status = 1 AND start_time <= ? AND end_time >= ?)", time.Now(), time.Now())
	}
	// 执行主查询
	err := query.Find(&results).Limit(20).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	// 收集地标ID
	pointIDs := make([]int64, len(results))
	for i, r := range results {
		pointIDs[i] = r.ID
	}

	// 使用子查询为每个地标找到第一张图片
	var images []model.CheckInPointImage
	subQuery := tx.Model(&model.CheckInPointImage{}).Select("MIN(id)").Where("check_in_point_id IN ?", pointIDs).Group("check_in_point_id")
	err = tx.Model(&model.CheckInPointImage{}).Where("id IN (?)", subQuery).Find(&images).Error
	if err != nil {
		return nil, err
	}

	// 创建地标ID到图片URL的映射
	imageMap := make(map[int64]string)
	for _, image := range images {
		imageMap[image.CheckInPointID] = image.ImageURL
	}

	// 将图片URL填充到结果中
	for _, result := range results {
		if imageUrl, ok := imageMap[result.ID]; ok {
			result.LandmarkImage = imageUrl
		}
	}

	return results, err
}
