package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	// AdvertisementStatusNormal 广告状态：正常
	AdvertisementStatusNormal = 1
	// AdvertisementStatusDisabled 广告状态：禁用
	AdvertisementStatusDisabled = 0
)

// AdvertisementListItem 广告列表项
type AdvertisementListItem struct {
	model.CheckInPointAdvertisement
	IsClaimed     bool               `json:"is_claimed"`     // 是否已领取
	Point         model.CheckInPoint `json:"point"`          // 地标信息
	OwnerUsername string             `json:"owner_username"` // 地标主人用户名
	OwnerAvatar   string             `json:"owner_avatar"`
	CategoryImage string             `json:"category_image"` // 分类图片
	Distance      float64            `json:"distance"`       // 距离（仅在按距离排序时返回）
	Images        []string           `json:"images"`         // 地标图片列表
}

// AdvertisementClaimRequest 广告领取请求参数
type AdvertisementClaimRequest struct {
	AdvertisementID int64 `json:"advertisement_id" form:"advertisement_id" binding:"required,min=1"` // 广告ID
}

// AdvertisementListByCategoryRequest 广告列表请求参数
type AdvertisementListByCategoryRequest struct {
	Category string `json:"category" form:"category"`                            // 分类，为空时展示全部
	SortBy   string `json:"sort_by" form:"sort_by"`                              // 排序方式：distance(距离)、reward(奖励)、time(时间)，默认distance
	Page     int    `json:"page" form:"page" binding:"required,min=1"`           // 页码
	PageSize int    `json:"page_size" form:"page_size" binding:"required,min=1"` // 每页数量
}

// AdvertisementListByCategoryResponse 广告列表响应
type AdvertisementListByCategoryResponse struct {
	Total int64                   `json:"total"` // 总数
	List  []AdvertisementListItem `json:"list"`  // 列表
}

// AdvertisementClaimResponse 广告领取响应
type AdvertisementClaimResponse struct {
	RewardAmount   decimal.Decimal `json:"reward_amount"`   // 奖励金额
	RewardCurrency string          `json:"reward_currency"` // 奖励单位
}

// advertisementListByCategory 获取分类广告列表
func advertisementListByCategory(ec *middleware.AppRequestContext) error {
	var req AdvertisementListByCategoryRequest
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("绑定请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Page > 1 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 获取用户位置信息
	var user model.User
	err := ec.Nu.DB.Model(&model.User{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&user).Error
	if err != nil {
		sdlog.Errorf("获取用户信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err = ec.Nu.DB.Model(&model.SpotCategory{}).Find(&categories).Error
	if err != nil {
		sdlog.Errorf("查询分类信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	categoryMap := make(map[string]*model.SpotCategory)
	for _, cat := range categories {
		categoryMap[cat.Type] = cat
	}

	// 构建基础查询
	query := ec.Nu.DB.Model(&model.CheckInPointAdvertisement{}).
		Select("check_in_point_advertisement.*, check_in_point.*, "+
			"6371 * 2 * ASIN(SQRT(POWER(SIN((? - check_in_point.latitude) * PI() / 180 / 2), 2) + "+
			"COS(? * PI() / 180) * COS(check_in_point.latitude * PI() / 180) * "+
			"POWER(SIN((? - check_in_point.longitude) * PI() / 180 / 2), 2))) as distance",
			user.Latitude, user.Latitude, user.Longitude).
		Joins("LEFT JOIN check_in_point ON check_in_point_advertisement.check_in_point_id = check_in_point.id").
		Where("check_in_point_advertisement.status = ?", AdvertisementStatusNormal).
		Where("check_in_point_advertisement.start_time <= ?", time.Now()).
		Where("check_in_point_advertisement.end_time >= ?", time.Now())

	// 如果指定了分类，添加分类过滤条件
	if req.Category != "" {
		query = query.Where("check_in_point.category_id = ?", req.Category)
	}

	// 根据排序方式添加排序条件
	switch req.SortBy {
	case "reward":
		query = query.Order("check_in_point_advertisement.reward_amount DESC")
	case "time":
		query = query.Order("check_in_point_advertisement.created_at DESC")
	default: // 默认按距离排序
		query = query.Order("distance ASC")
	}

	// 查询总数
	var total int64
	err = query.Count(&total).Error
	if err != nil {
		sdlog.Errorf("查询广告总数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询广告列表
	var results []struct {
		model.CheckInPointAdvertisement
		Point    model.CheckInPoint `gorm:"embedded"`
		Distance float64
	}
	err = query.Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&results).Error
	if err != nil {
		sdlog.Errorf("查询广告列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询用户领取记录
	var claimRecords []model.UserAdvertisementClaimRecord
	err = ec.Nu.DB.Model(&model.UserAdvertisementClaimRecord{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Find(&claimRecords).Error
	if err != nil {
		sdlog.Errorf("查询用户领取记录失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建广告ID到领取状态的映射
	claimedMap := make(map[int64]bool)
	for _, record := range claimRecords {
		claimedMap[record.AdvertisementID] = true
	}

	// 查询所有地标的图片
	var pointImages []model.CheckInPointImage
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN (?)", func() []int64 {
			ids := make([]int64, 0, len(results))
			for _, result := range results {
				ids = append(ids, result.CheckInPointID)
			}
			return ids
		}()).
		Find(&pointImages).Error
	if err != nil {
		sdlog.Errorf("查询地标图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建地标ID到图片列表的映射
	imageMap := make(map[int64][]string)
	for _, image := range pointImages {
		imageMap[image.CheckInPointID] = append(imageMap[image.CheckInPointID], image.ImageURL)
	}

	// 查询地标主人的信息
	var pointOwners []model.User
	err = ec.Nu.DB.Model(&model.User{}).
		Where("user_id IN (?)", func() []int64 {
			ids := make([]int64, 0, len(results))
			for _, result := range results {
				ids = append(ids, result.Point.OwnerID)
			}
			return ids
		}()).
		Find(&pointOwners).Error
	if err != nil {
		sdlog.Errorf("查询地标主人信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建用户ID到用户信息的映射
	ownerMap := make(map[int64]model.User)
	for _, owner := range pointOwners {
		ownerMap[owner.UserID] = owner
	}

	// 构建返回列表
	advertisements := make([]AdvertisementListItem, 0, len(results))
	for _, result := range results {
		owner := ownerMap[result.Point.OwnerID]
		item := AdvertisementListItem{
			CheckInPointAdvertisement: result.CheckInPointAdvertisement,
			IsClaimed:                 claimedMap[result.ID],
			Point:                     result.Point,
			OwnerUsername:             owner.Nickname,
			OwnerAvatar:               owner.AvatarURL,
			Distance:                  result.Distance,
			Images:                    imageMap[result.CheckInPointID],
			CategoryImage:             categoryMap[result.Point.Category].ImageURL,
		}
		advertisements = append(advertisements, item)
	}

	return webapi.OK(AdvertisementListByCategoryResponse{
		Total: total,
		List:  advertisements,
	}).Render(ec)
}

// advertisementClaim 领取广告
func advertisementClaim(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	return webapi.Error(sderr.Sentinel("coming soon")).Render(ec)
	// 使用Redis加锁
	lockKey := fmt.Sprintf("user_balance:lock:%d", userId)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	var req AdvertisementClaimRequest
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("绑定请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 开启事务
	tx := ec.Nu.DB.Begin()
	if tx.Error != nil {
		sdlog.Errorf("开启事务失败: %v", tx.Error)
		return webapi.Error(common.ErrService).Render(ec)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查广告是否存在且有效
	var advertisement model.CheckInPointAdvertisement
	err := tx.Model(&model.CheckInPointAdvertisement{}).
		Where("id = ?", req.AdvertisementID).
		Where("status = ?", AdvertisementStatusNormal).
		Where("start_time <= ?", time.Now()).
		Where("end_time >= ?", time.Now()).
		First(&advertisement).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("广告不存在或已失效: %v", err)
			return webapi.Error(common.ErrParam).Render(ec)
		}
		sdlog.Errorf("查询广告失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 将奖励金额转换为 decimal
	rewardAmount := decimal.NewFromFloat(advertisement.RewardAmount)

	// 检查用户是否已领取
	var count int64
	err = tx.Model(&model.UserAssetDetail{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Where("related_id = ?", req.AdvertisementID).
		Where("category = ?", common.UserAssetCategoryAdvertisement).
		Count(&count).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询领取记录失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if count > 0 {
		tx.Rollback()
		sdlog.Errorf("已领取奖励")
		return webapi.Error(common.ErrAllreadyClaimed).Render(ec)
	}

	// 检查用户资产记录是否存在
	var userAsset model.UserAsset
	err = tx.Model(&model.UserAsset{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Where("currency = ?", advertisement.RewardCurrency).
		First(&userAsset).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，创建新记录
			userAsset = model.UserAsset{
				UserID:    ec.AuthData.User.ID,
				Currency:  advertisement.RewardCurrency,
				Balance:   rewardAmount.InexactFloat64(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = tx.Create(&userAsset).Error
			if err != nil {
				tx.Rollback()
				sdlog.Errorf("创建用户资产记录失败: %v", err)
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			tx.Rollback()
			sdlog.Errorf("查询用户资产记录失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	} else {
		// 如果存在，更新余额
		userBalance := decimal.NewFromFloat(userAsset.Balance)
		newBalance := userBalance.Add(rewardAmount)
		err = tx.Model(&model.UserAsset{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Where("currency = ?", advertisement.RewardCurrency).
			Update("balance", newBalance.InexactFloat64()).Error
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("更新用户余额失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 记录领取信息
	record := model.UserAssetDetail{
		UserID:    ec.AuthData.User.ID,
		RelatedID: req.AdvertisementID,
		Category:  common.UserAssetCategoryAdvertisement,
		Rewards:   rewardAmount.InexactFloat64(),
		Currency:  advertisement.RewardCurrency,
	}
	err = tx.Create(&record).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("记录领取信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(AdvertisementClaimResponse{
		RewardAmount:   rewardAmount,
		RewardCurrency: advertisement.RewardCurrency,
	}).Render(ec)
}
