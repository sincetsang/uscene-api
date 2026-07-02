package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"gorm.io/gorm"
)

// spotList *** 是否可打卡及地标详情都在列表一块返回了。
func spotList(ec *middleware.AppRequestContext) error {
	radius := ec.QueryParams().Get("radius")
	radiusInt := sdparse.Int64Def(radius, 0)
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	isOwner := ec.QueryParams().Get("is_owner")
	keyword := ec.QueryParams().Get("keyword")
	like := ec.QueryParams().Get("like")
	likeInt := sdparse.Int64Def(like, 0)
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	userId := ec.AuthData.User.ID
	info, err := service.SpotList(ec.Request().Context(), ec.Nu, userId, radiusInt, latitude, longitude, isOwnerInt, keyword, likeInt)
	if err != nil {
		sdlog.Errorf("[spotList] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

// spotList *** 仅返回列表，地标详情单独返回
// 搜索店铺
func spotListV2(ec *middleware.AppRequestContext) error {
	radius := ec.QueryParams().Get("radius")
	radiusInt := sdparse.Int64Def(radius, 0)
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	isOwner := ec.QueryParams().Get("is_owner")
	keyword := ec.QueryParams().Get("keyword")
	sortBy := ec.QueryParams().Get("sort_by") // 排序方式：distance(距离)、reward(奖励)、time(时间)，默认distance
	hasAdvertisement := ec.QueryParams().Get("has_advertisement")
	hasAdvertisementInt := sdparse.Int64Def(hasAdvertisement, 0)
	category := ec.QueryParams().Get("category")
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	userId := ec.AuthData.User.ID
	info, err := service.SpotListV2(ec.Request().Context(), ec.Nu, userId, radiusInt, latitude, longitude, isOwnerInt, keyword, category, sortBy, hasAdvertisementInt)
	if err != nil {
		sdlog.Errorf("[spotListV2] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

type CheckInParam struct {
	SpotId int64 `json:"spot_id"`
}

func spotCheckIn(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := CheckInParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	info, err := service.CheckInAtPoint(ec.Request().Context(), ec.Nu, userId, req.SpotId)
	if err != nil {
		sdlog.Errorf("[spotCheckIn] 打卡失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

type SpotCategory struct {
	Type     string `json:"type"`
	Label    string `json:"label"`
	ImageUrl string `json:"image_url"`
}

func spotCategory(ec *middleware.AppRequestContext) error {
	var result = []SpotCategory{}
	var list []*model.SpotCategory
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SpotCategory{}).Where("display = 1").Order("sort_index").Find(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("[spotCategory] 查询地标分类失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	lan := common.GetRequestLanguage()

	for _, item := range list {
		label := item.Label
		if lan == common.LanguageZh {
			label = item.LabelZh
		} else if lan == common.LanguageZhTw {
			label = item.LabelZhTw
		} else if lan == common.LanguageRu {
			label = item.LabelRu
		}
		if label == "" {
			label = item.Label
		}
		category := SpotCategory{
			Type:     item.Type,
			Label:    label,
			ImageUrl: item.ImageURL,
		}
		result = append(result, category)
	}
	return webapi.OK(result).Render(ec)
}

// 创建地标请求参数
type CreateSpotRequest struct {
	Title     string   `json:"title"`      // 地标名称
	Latitude  float64  `json:"latitude"`   // 纬度
	Longitude float64  `json:"longitude"`  // 经度
	Location  string   `json:"location"`   // 详细地址
	ImageUrls []string `json:"image_urls"` // 图片URL列表
	Category  string   `json:"category"`   // 分类
	// Country     string   `json:"country"`     // 国家
	// City        string `json:"city"`        // 城市
	// Province    string `json:"province"`    // 省份
	CheckCode   string `json:"check_code"`  // 核销码
	Description string `json:"description"` // 描述
	ExternalUrl string `json:"external_url"`
	TelegramUrl string `json:"telegram_url"`
	BelieveUrl  string `json:"believe_url"`
	WhatsappUrl string `json:"whatsapp_url"`
	PcnUrl      string `json:"pcn_url"`
	Wechat      string `json:"wechat"`
	BelieveId   string `json:"believe_id"`
}

func spotCreate(ec *middleware.AppRequestContext) error {
	// 开启事务
	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	if tx.Error != nil {
		sdlog.Errorf("[spotCreate] 开启事务失败: %v", tx.Error)
		return webapi.Error(common.ErrService).Render(ec)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 解析请求参数
	req := CreateSpotRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("[spotCreate] 解析请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 2. 参数验证
	// if req.Title == "" || req.Location == "" || len(req.ImageUrls) == 0 || req.Category == "" {
	if req.Title == "" || len(req.ImageUrls) == 0 || req.Category == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if len(req.Title) > 50 {
		return webapi.Error(common.ErrSpotNameLength).Render(ec)
	}

	country, province, city, _, err := service.GetAddressInfo(ec.Nu.Config.GoogleMapsKey, req.Latitude, req.Longitude)
	if err != nil {
		sdlog.Errorf("[spotCreate] 获取地址信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 3. 检查分类是否有效
	var category model.SpotCategory
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("type = ? AND display = 1", req.Category).
		First(&category).Error
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// err = service.CheckAddSpotPower(ec.Request().Context(), ec.Nu, ec.AuthData.User.ID, req.Latitude, req.Longitude)
	// if err != nil {
	// 	return webapi.Error(err).Render(ec)
	// }

	// 查询核销码是否已被使用
	req.CheckCode = strings.TrimSpace(req.CheckCode)
	var existingSpot model.VerificationCode
	err = tx.Where("code = ?", req.CheckCode).First(&existingSpot).Error
	if err == nil {
		if existingSpot.Status == common.VerificationCodeStatusUsed {
			tx.Rollback()
			return webapi.Error(fmt.Errorf("该核销码已被使用")).Render(ec)
		}
		// 更新核销码状态为已使用
		existingSpot.Status = common.VerificationCodeStatusUsed
		existingSpot.UseTime = time.Now()
		if err = tx.Save(&existingSpot).Error; err != nil {
			tx.Rollback()
			sdlog.Errorf("[spotCreate] 更新核销码状态失败: %v", err)
			return webapi.Error(fmt.Errorf("更新核销码状态失败")).Render(ec)
		}
	} else {
		// 1. 调用 UE API 验证核销码
		resp, err := ue_api_v2.UpdateTCodeStatus(models.TCodeReq{
			UEReqbaseV2: models.NewUEReqbaseV2(),
			Code:        req.CheckCode},
			ec.Nu.UeConfigParam)
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("[spotCreate] 验证核销码失败: %v", err)
			return webapi.Error(fmt.Errorf("验证核销码失败")).Render(ec)
		}
		if resp.Result <= 0 {
			tx.Rollback()
			sdlog.Errorf("[spotCreate] 核销码无效: %s", resp.Message)
			return webapi.Error(fmt.Errorf("核销码无效")).Render(ec)
		}
	}

	if req.TelegramUrl != "" && !(strings.Contains(req.TelegramUrl, "https://t.me") || strings.Contains(req.TelegramUrl, "http://t.me")) {
		req.TelegramUrl = "https://t.me/" + req.TelegramUrl
	}

	if req.BelieveUrl != "" && !(strings.Contains(req.BelieveUrl, "bmc://conversation")) {
		req.BelieveUrl = "bmc://conversation?type=private&targetId=" + req.BelieveUrl
	}

	if req.WhatsappUrl != "" && !(strings.Contains(req.WhatsappUrl, "https://wa.me/")) {
		req.WhatsappUrl = "https://wa.me/" + req.WhatsappUrl
	}

	if req.PcnUrl != "" && !(strings.Contains(req.PcnUrl, "p://user")) {
		req.PcnUrl = "p://user?address=" + req.PcnUrl
	}

	// 2. 创建地标记录
	spot := &model.CheckInPoint{
		Title:    req.Title,
		TitleCn:  "", // 初始值设为空字符串
		TitleEn:  "", // 初始值设为空字符串
		Category: req.Category,
		Currency: "platform",
		Level:    1,
		// LandmarkImage:    fmt.Sprintf("%s/%s", ec.Nu.Config.Aws.AwsS3CustomDomain, req.ImageUrls[0]), // 使用第一张图片作为主图
		Location:         req.Location,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		TotalCheckedIn:   0,
		Status:           1,
		HashRate:         1.0,
		Balance:          0,
		OwnerID:          ec.AuthData.User.ID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Country:          country,
		City:             city,
		Region:           province,
		Description:      req.Description,
		DescriptionCn:    "", // 初始值设为空字符串
		DescriptionEn:    "", // 初始值设为空字符串
		VerificationCode: req.CheckCode,
		IsActive:         1,
		ExternalURL:      req.ExternalUrl,
		AuditStatus:      common.SpotAuditStatusPending,
		TelegramURL:      req.TelegramUrl,
		BelieveURL:       req.BelieveUrl,
		WhatsappURL:      req.WhatsappUrl,
		PcnURL:           req.PcnUrl,
		Wechat:           req.Wechat,
		BelieveID:        req.BelieveId,
		// AuditStatus: common.SpotAuditStatusApproved,
	}

	if err = tx.Create(spot).Error; err != nil {
		tx.Rollback()
		sdlog.Errorf("[spotCreate] 创建地标失败: %v", err)
		return webapi.Error(fmt.Errorf("创建地标失败")).Render(ec)
	}

	// 创建图片记录
	if err = service.CreateUserImages(ec.Request().Context(), ec.Nu, tx, ec.AuthData.User.ID, req.ImageUrls, spot.ID); err != nil {
		tx.Rollback()
		sdlog.Errorf("[spotCreate] 创建图片记录失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if err = tx.Commit().Error; err != nil {
		sdlog.Errorf("[spotCreate] 提交事务失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 在事务提交后执行异步检查
	for _, fileKey := range req.ImageUrls {
		go service.AsyncCheckImageContent(context.Background(), ec.Nu, fileKey)
	}

	// 异步检查 spot 的图片状态
	go service.AsyncCheckSpotImageStatus(context.Background(), ec.Nu, spot.ID)

	// 查询用户信息
	var user model.User
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&user).Error
	if err != nil {
		sdlog.Errorf("[spotCreate] 查询用户信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建返回数据
	type SpotResponse struct {
		model.CheckInPoint
		OwnerName   string `json:"owner_name"`
		OwnerAvatar string `json:"owner_avatar"`
	}

	response := &SpotResponse{
		CheckInPoint: *spot,
		OwnerName:    user.Nickname,
		OwnerAvatar:  user.AvatarURL,
	}

	// 异步处理翻译
	go service.TranslateSpotContent(ec.Nu.OpenaiApiKey, spot.ID, req.Title, req.Description)

	sdlog.Infof("创建地标成功，地标ID: %d", spot.ID)

	return webapi.OK(response).Render(ec)
}

// 收藏地标请求参数
type SpotLikeRequest struct {
	SpotID int64 `json:"spot_id"` // 地标ID
}

// 收藏地标
func spotLike(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := SpotLikeRequest{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 参数验证
	if req.SpotID <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查地标是否存在
	var spot model.CheckInPoint
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("id = ? AND status = 1", req.SpotID).
		First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrSpotNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查是否已收藏
	var existingLike model.UserLike
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ? AND spot_id = ?", userId, req.SpotID).
		First(&existingLike).Error

	if err == nil {
		// 已收藏，则取消收藏
		err = ec.Nu.DB.WithContext(ec.Request().Context()).
			Delete(&existingLike).Error
		if err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}

		// 更新地标收藏数
		likeCount := spot.LikeCount - 1
		if likeCount < 0 {
			likeCount = 0
		}
		_ = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.CheckInPoint{}).Where("id = ?", req.SpotID).Update("like_count", likeCount).Error

		return webapi.OK(map[string]interface{}{
			"is_liked": false,
		}).Render(ec)
	}

	if err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}
	redis.AddClaim(ec.Request().Context(), ec.Nu, spot.OwnerID, fmt.Sprint(spot.ID), 10)
	// 创建收藏记录
	like := &model.UserLike{
		UserID:    userId,
		SpotID:    req.SpotID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Create(like).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 更新地标收藏数
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.CheckInPoint{}).Where("id = ?", req.SpotID).Update("like_count", gorm.Expr("like_count + 1")).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(map[string]interface{}{
		"is_liked": true,
	}).Render(ec)
}

// 获取用户收藏的地标列表
func spotLikeList(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 获取分页参数
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}

	pageNum, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeNum, err := strconv.ParseInt(pageSize, 10, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 定义一个临时结构体来接收基础查询结果
	type BaseResult struct {
		model.CheckInPoint
		Ownername   string `gorm:"column:owner_name" json:"owner_name"`
		OwnerAvatar string `gorm:"column:owner_avatar" json:"owner_avatar"`
	}

	var baseResults []BaseResult

	// 查询总数
	var total int64
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("check_in_point").
		Select("COUNT(DISTINCT check_in_point.id)").
		Joins("INNER JOIN user_like ON check_in_point.id = user_like.spot_id").
		Where("user_like.user_id = ? AND check_in_point.status = 1", userId).
		Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询收藏的地标列表
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("check_in_point").
		Select("check_in_point.*, user.nickname as owner_name, user.avatar_url as owner_avatar").
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Joins("INNER JOIN user_like ON check_in_point.id = user_like.spot_id").
		Where("user_like.user_id = ? AND check_in_point.status = 1", userId).
		Order("user_like.created_at DESC").
		Offset(int((pageNum - 1) * pageSizeNum)).
		Limit(int(pageSizeNum)).
		Find(&baseResults).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if len(baseResults) == 0 {
		return webapi.OK(MySpotListResponse{
			TotalCount: 0,
			Points:     []*CheckInPointWithOwner{},
		}).Render(ec)
	}
	// 获取所有地标ID
	var spotIDs []int64
	for _, spot := range baseResults {
		spotIDs = append(spotIDs, spot.ID)
	}

	// 批量查询图片
	var images []model.CheckInPointImage
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ?", spotIDs).
		Order("check_in_point_id, sort_index").
		Find(&images).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建图片映射
	spotImages := make(map[int64][]model.CheckInPointImage)
	for _, img := range images {
		spotImages[img.CheckInPointID] = append(spotImages[img.CheckInPointID], img)
	}

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SpotCategory{}).Find(&categories).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	categoryMap := make(map[string]*model.SpotCategory)
	for _, cat := range categories {
		categoryMap[cat.Type] = cat
		categoryMap[cat.Label] = cat
		categoryMap[cat.LabelZh] = cat
		categoryMap[cat.LabelZhTw] = cat
	}

	// 构建响应数据
	points := make([]*CheckInPointWithOwner, len(baseResults))
	for i, spot := range baseResults {
		point := &CheckInPointWithOwner{
			Point:         &spot.CheckInPoint,
			OwnerUsername: spot.Ownername,
			OwnerAvatar:   spot.OwnerAvatar,
			Liked:         true,
			Images:        make([]model.CheckInPointImage, 0),
		}

		// 添加其他图片
		if imgs, exists := spotImages[spot.ID]; exists {
			point.Images = append(point.Images, imgs...)
		}
		// 添加主图
		if spot.LandmarkImage != "" && len(point.Images) == 0 {
			point.Images = append(point.Images, model.CheckInPointImage{
				ID:             0,
				CheckInPointID: spot.ID,
				ImageURL:       spot.LandmarkImage,
				SortIndex:      0,
				Status:         1,
			})
		}
		point.CategoryImage = categoryMap[spot.Category].ImageURL
		// 设置分类名称
		lan := common.GetRequestLanguage()
		if lan == common.LanguageZh {
			point.Point.Category = categoryMap[spot.Category].LabelZh
		} else if lan == common.LanguageZhTw {
			point.Point.Category = categoryMap[spot.Category].LabelZhTw
		} else if lan == common.LanguageRu {
			point.Point.Category = categoryMap[spot.Category].LabelRu
		}

		points[i] = point
	}

	return webapi.OK(MySpotListResponse{
		TotalCount: total,
		Points:     points,
	}).Render(ec)
}

type SpotCliamParam struct {
	SpotId int64 `json:"spot_id"`
}

type SpotClaimRes struct {
	Claim float64 `json:"claim"`
}

func spotClaim(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := SpotCliamParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.SpotId <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	claim, err := redis.Claim(ec.Request().Context(), ec.Nu, userId, fmt.Sprint(req.SpotId))
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(SpotClaimRes{
		Claim: claim,
	}).Render(ec)
}

// 获取地标详情
func spotDetail(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 从 URL 查询参数获取 spot_id
	spotID := ec.QueryParams().Get("spot_id")
	if spotID == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 转换为 int64
	spotIDInt, err := strconv.ParseInt(spotID, 10, 64)
	if err != nil || spotIDInt <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// 查询地标信息
	var spot db.CheckInPointWithOwnerV2
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.CheckInPoint{}).
		Select(`
			DISTINCT check_in_point.*,
			user.nickname as owner_username,
			user.avatar_url as owner_avatar,
			CASE WHEN user_like.id IS NOT NULL THEN 1 ELSE 0 END as liked
		`).
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Joins("LEFT JOIN user_like ON check_in_point.id = user_like.spot_id AND user_like.user_id = ?", userId).
		Where("check_in_point.id = ? AND check_in_point.status = 1 and check_in_point.is_active=1 and check_in_point.audit_status=1", spotIDInt).
		First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrSpotNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	if spot.TelegramURL != "" {
		spot.TelegramUid = strings.TrimPrefix(spot.TelegramURL, "https://t.me/")
	}
	if spot.WhatsappURL != "" {
		spot.WhatsappUid = strings.TrimPrefix(spot.WhatsappURL, "https://wa.me/")
	}
	if spot.PcnURL != "" {
		spot.PcnUid = strings.TrimPrefix(spot.PcnURL, "p://user?address=")
	}

	// 查询地标图片
	var images []model.CheckInPointImage
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("check_in_point_id = ?", spotIDInt).
		Where("status = 1").
		Order("sort_index").
		Find(&images).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if spot.LandmarkImage == "" && len(images) > 0 {
		spot.LandmarkImage = images[0].ImageURL
	}

	// 查询分类信息
	var category model.SpotCategory
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("type = ? or label = ? or label_zh = ?", spot.Category, spot.Category, spot.Category).
		First(&category).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询广告信息
	var advertisement *model.CheckInPointAdvertisement
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("check_in_point_id = ? AND status = 1 AND start_time <= ? AND end_time >= ?", spotIDInt, time.Now(), time.Now()).
		First(&advertisement).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrService).Render(ec)
		}
		// 如果没有找到广告，将 advertisement 设置为 nil
		advertisement = nil
	}

	// 查询商品信息
	var products []model.Product
	productQuery := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("check_in_point_id = ?", spotIDInt)
	if spot.OwnerID != userId {
		productQuery = productQuery.Where("audit_status = 1 and is_active = 1")
	}
	err = productQuery.Find(&products).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查广告是否已领取
	advertisementIsClaimed := false
	if advertisement != nil {
		var count int64
		err = ec.Nu.DB.WithContext(ec.Request().Context()).
			Model(&model.UserAssetDetail{}).
			Where("user_id = ?", userId).
			Where("related_id = ?", advertisement.ID).
			Where("category = ?", common.UserAssetCategoryAdvertisement).
			Count(&count).Error
		if err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
		advertisementIsClaimed = count > 0
	}

	// 计算可领取奖励
	claim := 0.0
	if advertisement != nil && !advertisementIsClaimed {
		claim = advertisement.RewardAmount
	}

	lanCategory := category.Label
	lan := common.GetRequestLanguage()
	if lan == common.LanguageZh {
		lanCategory = category.LabelZh
	} else if lan == common.LanguageZhTw {
		lanCategory = category.LabelZhTw
	}

	productIds := make([]int64, 0)
	for _, p := range products {
		productIds = append(productIds, int64(p.ID))
	}
	productImageMap, err := service.GetProductImageMapByProductIds(ec.Request().Context(), ec.Nu, productIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	productsWithImage := make([]service.ProductInfoWithImage, 0)
	for _, p := range products {
		if productImages, exists := productImageMap[int64(p.ID)]; exists {
			productsWithImage = append(productsWithImage, service.ProductInfoWithImage{
				Product: p,
				HeadImg: productImages[0].ImageURL,
				Images:  productImages,
			})
		} else {
			productsWithImage = append(productsWithImage, service.ProductInfoWithImage{
				Product: p,
				HeadImg: "",
				Images:  make([]model.ProductImage, 0),
			})
		}
	}

	minPrice := 0.0
	minPriceCurrency := ""
	for _, p := range products {
		if p.AmountUsd > 0 {
			minPrice = p.AmountUsd
			minPriceCurrency = "USDT"
			break
		} else if p.Amount > 0 {
			minPrice = p.Amount
			minPriceCurrency = "FEC"
			break
		}
	}

	minPriceStr := ""
	// 有小数的保留两位小数，没小数的保留整数
	if minPrice == math.Trunc(minPrice) {
		minPriceStr = fmt.Sprintf("%.0f", minPrice)
	} else {
		minPriceStr = fmt.Sprintf("%.2f", minPrice)
	}

	// 构建返回数据
	response := &service.CheckInPointStruct{
		CategoryImage:          category.ImageURL,
		Claim:                  claim,
		AdvertisementIsClaimed: true,
		Point:                  &spot,
		PointImages:            images,
		Advertisement:          advertisement,
		Product:                productsWithImage,
		Category:               lanCategory,
		CategoryEn:             category.Type,
		MinPrice:               minPriceStr,
		MinPriceCurrency:       minPriceCurrency,
	}

	return webapi.OK(response).Render(ec)
}

type SpotAddressRes struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Address  string `json:"address"`
}

func spotAddress(ec *middleware.AppRequestContext) error {
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	if latitude == "" || longitude == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	latitudeFloat, err := strconv.ParseFloat(latitude, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	longitudeFloat, err := strconv.ParseFloat(longitude, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	country, province, city, address, err := service.GetAddressInfo(ec.Nu.Config.GoogleMapsKey, latitudeFloat, longitudeFloat)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(SpotAddressRes{
		Country:  country,
		Province: province,
		City:     city,
		Address:  address,
	}).Render(ec)
}

type UpdateSpotRequest struct {
	SpotID    int64    `json:"spot_id"`
	Title     string   `json:"title" binding:"required"`
	Location  string   `json:"location" binding:"required"`
	ImageUrls []string `json:"image_urls" binding:"required"`
	Category  string   `json:"category" binding:"required"`
	Latitude  float64  `json:"latitude" binding:"required"`
	Longitude float64  `json:"longitude" binding:"required"`
	// Country        string   `json:"country"`
	// City           string   `json:"city"`
	// Province       string   `json:"province"`
	Description    string  `json:"description"`
	ExternalUrl    string  `json:"external_url"`
	TelegramUrl    string  `json:"telegram_url"`
	DeleteImageIds []int64 `json:"delete_image_ids"`
	BelieveUrl     string  `json:"believe_url"`
	WhatsappUrl    string  `json:"whatsapp_url"`
	PcnUrl         string  `json:"pcn_url"`
	Wechat         string  `json:"wechat"`
	BelieveId      string  `json:"believe_id"`
}

func spotUpdate(ec *middleware.AppRequestContext) error {
	// 开启事务
	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	if tx.Error != nil {
		sdlog.Errorf("[spotUpdate] 开启事务失败: %v", tx.Error)
		return webapi.Error(common.ErrService).Render(ec)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 2. 解析请求参数
	req := UpdateSpotRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("[spotUpdate] 解析请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 3. 参数验证
	if req.Title == "" || req.Category == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if len(req.Title) > 50 {
		return webapi.Error(common.ErrSpotNameLength).Render(ec)
	}

	// 4. 检查分类是否有效
	var category model.SpotCategory
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("(type = ? OR label = ? or label_zh=? or label_zh_tw=?) AND display = 1", req.Category, req.Category, req.Category, req.Category).
		First(&category).Error
	if err != nil {
		sdlog.Errorf("[spotUpdate] 分类不存在: %v", req.Category)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 5. 查询地标是否存在
	var spot model.CheckInPoint
	err = tx.Where("id = ?", req.SpotID).First(&spot).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("[spotUpdate] 地标不存在: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 6. 检查权限
	if spot.OwnerID != ec.AuthData.User.ID {
		tx.Rollback()
		return webapi.Error(common.ErrParam).Render(ec)
	}

	country, province, city, _, err := service.GetAddressInfo(ec.Nu.Config.GoogleMapsKey, req.Latitude, req.Longitude)
	if err != nil {
		sdlog.Errorf("[spotCreate] 获取地址信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if req.TelegramUrl != "" && !(strings.Contains(req.TelegramUrl, "https://t.me") || strings.Contains(req.TelegramUrl, "http://t.me")) {
		req.TelegramUrl = "https://t.me/" + req.TelegramUrl
	}

	if req.BelieveUrl != "" && !(strings.Contains(req.BelieveUrl, "bmc://conversation")) {
		req.BelieveUrl = "bmc://conversation?type=private&targetId=" + req.BelieveUrl
	}

	if req.WhatsappUrl != "" && !(strings.Contains(req.WhatsappUrl, "https://wa.me/")) {
		req.WhatsappUrl = "https://wa.me/" + req.WhatsappUrl
	}

	if req.PcnUrl != "" && !(strings.Contains(req.PcnUrl, "p://user")) {
		req.PcnUrl = "p://user?address=" + req.PcnUrl
	}

	// 7. 更新地标信息
	updates := map[string]interface{}{
		"title":        req.Title,
		"category":     req.Category,
		"location":     req.Location,
		"latitude":     req.Latitude,
		"longitude":    req.Longitude,
		"country":      country,
		"city":         city,
		"region":       province,
		"description":  req.Description,
		"external_url": req.ExternalUrl,
		"telegram_url": req.TelegramUrl,
		"believe_url":  req.BelieveUrl,
		"whatsapp_url": req.WhatsappUrl,
		"pcn_url":      req.PcnUrl,
		"wechat":       req.Wechat,
		"believe_id":   req.BelieveId,
		"updated_at":   time.Now(),
		"audit_status": common.SpotAuditStatusPending, // 修改后需要重新审核
	}
	needTranslate := false
	if req.Title != spot.Title {
		updates["title_cn"] = ""
		updates["title_en"] = ""
		needTranslate = true
	}
	if req.Description != spot.Description {
		updates["description_cn"] = ""
		updates["description_en"] = ""
		needTranslate = true
	}

	if err = tx.Model(&spot).Updates(updates).Error; err != nil {
		tx.Rollback()
		sdlog.Errorf("[spotUpdate] 更新地标失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 先删除旧的图片记录
	if len(req.DeleteImageIds) > 0 {
		if err = tx.Where("check_in_point_id = ? and id in (?)", spot.ID, req.DeleteImageIds).Delete(&model.CheckInPointImage{}).Error; err != nil {
			tx.Rollback()
			sdlog.Errorf("[spotUpdate] 删除旧图片记录失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 创建新的图片记录
	if len(req.ImageUrls) > 0 {
		if err = service.CreateUserImages(ec.Request().Context(), ec.Nu, tx, ec.AuthData.User.ID, req.ImageUrls, spot.ID); err != nil {
			tx.Rollback()
			sdlog.Errorf("[spotUpdate] 创建图片记录失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	if err = tx.Commit().Error; err != nil {
		sdlog.Errorf("[spotUpdate] 提交事务失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if len(req.ImageUrls) > 0 {
		// 在事务提交后执行异步检查
		for _, fileKey := range req.ImageUrls {
			go service.AsyncCheckImageContent(context.Background(), ec.Nu, fileKey)
		}
	}
	// 异步检查 spot 的图片状态
	go service.AsyncCheckSpotImageStatus(context.Background(), ec.Nu, spot.ID)

	if needTranslate {
		// 异步处理翻译
		go service.TranslateSpotContent(ec.Nu.OpenaiApiKey, spot.ID, req.Title, req.Description)
	}

	// 查询用户信息
	var user model.User
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&user).Error
	if err != nil {
		sdlog.Errorf("[spotUpdate] 查询用户信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建返回数据
	type SpotResponse struct {
		model.CheckInPoint
		OwnerName   string `json:"owner_name"`
		OwnerAvatar string `json:"owner_avatar"`
	}

	response := &SpotResponse{
		CheckInPoint: spot,
		OwnerName:    user.Nickname,
		OwnerAvatar:  user.AvatarURL,
	}

	sdlog.Infof("更新地标成功，地标ID: %d", spot.ID)

	return webapi.OK(response).Render(ec)
}
