package db

import (
	"TMA/service/db/model"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func GetSpotLevelAll(tx *gorm.DB) ([]*model.SpotLevelConfig, error) {
	var list []*model.SpotLevelConfig
	err := tx.Model(&model.SpotLevelConfig{}).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

type CheckInPointWithOwner struct {
	model.CheckInPoint
	OwnerUsername          string                          `json:"owner_username"`
	OwnerAvatar            string                          `json:"owner_avatar"`
	Liked                  bool                            `json:"liked"`
	CategoryImage          string                          `json:"category_image"`
	Images                 []model.CheckInPointImage       `gorm:"-"  json:"images"`
	Claim                  float64                         `json:"claim"`
	Product                []model.Product                 `gorm:"-"  json:"product"`
	Advertisement          model.CheckInPointAdvertisement `json:"advertisement" gorm:"check_in_point_advertisement"` // 广告详情
	AdvertisementIsClaimed bool                            `json:"advertisement_is_claimed" gorm:"-"`                 // 广告是否已领取
}

func GetAllValidCheckPointsByLaLo(tx *gorm.DB, latMin, latMax, lonMin, lonMax float64, OwnerId int64, keyword string, like_uid int64) ([]*CheckInPointWithOwner, error) {
	var results []*CheckInPointWithOwner

	// 构建基础查询
	query := tx.Model(&model.CheckInPoint{}).
		Select(`
			DISTINCT check_in_point.*,
			user.nickname as owner_username,
			user.avatar_url as owner_avatar,
			CASE WHEN user_like.id IS NOT NULL THEN 1 ELSE 0 END as liked
		`).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Joins("LEFT JOIN user_like ON check_in_point.id = user_like.spot_id AND user_like.user_id = ?", like_uid).
		Where("check_in_point.latitude >= ? AND check_in_point.latitude <= ? AND check_in_point.longitude >= ? AND check_in_point.longitude <= ? AND check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1",
			latMin, latMax, lonMin, lonMax)

	// 添加条件查询
	if OwnerId != 0 {
		query = query.Where("owner_id = ?", OwnerId)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%").Limit(30)
	}

	// 执行主查询
	err := query.Find(&results).Error
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
	spotIds := make([]int64, len(results))
	spotMap := make(map[int64]*CheckInPointWithOwner, len(results))
	for i, spot := range results {
		spotIds[i] = spot.ID
		spotMap[spot.ID] = spot
		spot.Images = make([]model.CheckInPointImage, 0)
	}

	// 批量查询产品信息
	var products []model.Product
	err = tx.Model(&model.Product{}).
		Where("check_in_point_id IN ?", spotIds).
		Order("check_in_point_id, level").
		Find(&products).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 分配产品到对应的地标
	for _, product := range products {
		if spot, exists := spotMap[product.CheckInPointID]; exists {
			spot.Product = append(spot.Product, product)
		}
	}
	// 批量查询图片
	var images []model.CheckInPointImage
	err = tx.Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ?", spotIds).
		Order("check_in_point_id, sort_index").
		Find(&images).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 添加其他图片
	for _, img := range images {
		if spot, exists := spotMap[img.CheckInPointID]; exists {
			spot.Images = append(spot.Images, img)
		}
	}

	// 分配图片到对应的地标
	for _, spot := range results {
		if spot.LandmarkImage != "" && len(spot.Images) == 0 {
			spot.Images = append(spot.Images, model.CheckInPointImage{
				ID:             0,
				CheckInPointID: spot.ID,
				ImageURL:       spot.LandmarkImage,
				SortIndex:      0,
				Status:         1,
			})
		}
	}

	// 查询广告领取记录
	var claimRecords []model.UserAdvertisementClaimRecord
	err = tx.Model(&model.UserAdvertisementClaimRecord{}).
		Where("user_id = ?", like_uid).
		Find(&claimRecords).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 构建广告ID到领取状态的映射
	claimedMap := make(map[int64]bool)
	for _, record := range claimRecords {
		claimedMap[record.AdvertisementID] = true
	}

	// 为每个地标单独查询广告信息
	for _, spot := range results {
		var advertisement model.CheckInPointAdvertisement
		err = tx.Model(&model.CheckInPointAdvertisement{}).
			Where("check_in_point_id = ? AND status = 1", spot.ID).
			First(&advertisement).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		spot.Advertisement = advertisement

		// 设置广告领取状态
		if spot.Advertisement.ID > 0 && spot.Advertisement.Status == 1 {
			spot.AdvertisementIsClaimed = claimedMap[spot.Advertisement.ID]
		} else {
			spot.AdvertisementIsClaimed = false
		}
	}

	return results, nil
}

func GetUserCheckPointsByUserIdAndDate(tx *gorm.DB, uid int64, date string) ([]*model.UserCheckRecord, error) {
	result := []*model.UserCheckRecord{}
	err := tx.Model(&model.UserCheckRecord{}).Where("user_id = ? and day = ?", uid, date).Find(&result).Error
	if err == gorm.ErrRecordNotFound {
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

type UserSpotChecked struct {
	PointId int64 `json:"point_id"`
}

func SpotUserCheckedByIds(tx *gorm.DB, uid int64, pointIds []int64) ([]UserSpotChecked, error) {
	todayDate := time.Now().Format("20060102")
	var res []UserSpotChecked
	err := tx.Model((&model.UserCheckRecord{})).
		Select("point_id").
		Where("user_id = ?", uid).
		Where("point_id in (?) and day = ?", pointIds, todayDate).
		Scan(&res).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		fmt.Printf(`SpotUserCheckedByIds err %s, %d\n\n`, err.Error(), len(pointIds))
		return nil, err
	}
	return res, nil
}

func GetValidCheckPointById(tx *gorm.DB, id int64) (*model.CheckInPoint, error) {
	result := &model.CheckInPoint{}
	err := tx.Model(&model.CheckInPoint{}).Where("id = ? and status=1", id).First(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetSpotLevelInfoByLevel(tx *gorm.DB, level int64) (*model.SpotLevelConfig, error) {
	var info model.SpotLevelConfig
	err := tx.Model(&model.SpotLevelConfig{}).Where("level = ?", level).First(&info).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}
