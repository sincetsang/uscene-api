package main

import (
	"TMA/pkg/cache/memory"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db"
	"TMA/service/db/model"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"github.com/sulink/ueapisdk/utils/constants"
	"gorm.io/gorm"
)

// ProductListResponse 产品列表响应结构体
type ProductListResponse struct {
	Products []ProductInfo `json:"products"` // 产品列表
}

type ProductListResponseWithDistance struct {
	Products []ProductInfoWithDistance `json:"products"` // 产品列表
}

type ProductImage struct {
	ID        int64  `json:"id"`
	ImageURL  string `json:"image_url"`
	Status    int32  `json:"status"`
	ImageType string `json:"image_type"`
}

type ProductTag struct {
	TagCn     string `json:"tag_cn"`
	TagEn     string `json:"tag_en"`
	SortIndex int32  `json:"sort_index"`
	Mark      string `json:"mark"`
}

// ProductInfo 产品信息结构体
type ProductInfo struct {
	SpotID            int64  `json:"spot_id"`             // 地标ID
	SpotTitleCn       string `json:"spot_title_cn"`       // 地标名称
	SpotTitleEn       string `json:"spot_title_en"`       // 地标名称
	SpotDescriptionCn string `json:"spot_description_cn"` // 地标描述
	SpotDescriptionEn string `json:"spot_description_en"` // 地标描述
	SpotImage         string `json:"spot_image"`          // 地标图片
	SpotCategory      string `json:"spot_category"`       // 地标分类
	SpotCategoryEn    string `json:"spot_category_en"`    // 地标分类英文
	SpotCategoryImage string `json:"spot_category_image"` // 地标分类图片
	OwnerId           int64  `json:"owner_id"`            // 地标拥有者ID

	ProductID     int64          `json:"product_id"`                   // 产品ID
	Name          string         `json:"name"`                         // 产品名称
	Description   string         `json:"description"`                  // 产品描述
	Amount        float64        `json:"amount"`                       // 产品价格
	AmountUsd     float64        `json:"amount_usd"`                   // 产品价格（USDT）
	CheckCodeNum  int32          `json:"check_code_num"`               // 检查码数量
	HeadImg       string         `gorm:"column:image" json:"head_img"` // 产品头图
	Images        []ProductImage `json:"images"`                       // 产品图片列表
	CommentCount  int64          `json:"comment_count"`                // 评论数量
	VisitedCount  int64          `json:"visited_count"`                // 浏览次数
	Tags          []ProductTag   `json:"tags"`                         // 产品标签
	FavoriteCount int64          `json:"favorite_count"`               // 收藏次数
	AuditStatus   int32          `json:"audit_status"`                 // 审核状态
	IsActive      bool           `json:"is_active"`                    // 是否激活
	//联系方式
	Wechat      string `json:"wechat"` // 联系方式
	BelieveURL  string `gorm:"column:believe_url;not null" json:"believe_url"`
	PcnURL      string `gorm:"column:pcn_url;not null" json:"pcn_url"`
	WhatsappURL string `gorm:"column:whatsapp_url;not null" json:"whatsapp_url"`
	BelieveID   string `gorm:"column:believe_id;not null" json:"believe_id"`
	TelegramURL string `gorm:"column:telegram_url;not null" json:"telegram_url"`
}

type ProductInfoWithOutSpotInfo struct {
	SpotID        int64          `json:"spot_id"`                      // 地标ID
	ProductID     int64          `json:"product_id"`                   // 产品ID
	Name          string         `json:"name"`                         // 产品名称
	Description   string         `json:"description"`                  // 产品描述
	Amount        float64        `json:"amount"`                       // 产品价格
	AmountUsd     float64        `json:"amount_usd"`                   // 产品价格（USDT）
	CheckCodeNum  int32          `json:"check_code_num"`               // 检查码数量
	HeadImg       string         `gorm:"column:image" json:"head_img"` // 产品头图
	Images        []ProductImage `json:"images"`                       // 产品图片列表
	CommentCount  int64          `json:"comment_count"`                // 评论数量
	VisitedCount  int64          `json:"visited_count"`                // 浏览次数
	Tags          []ProductTag   `json:"tags"`                         // 产品标签
	FavoriteCount int64          `json:"favorite_count"`               // 收藏次数
	AuditStatus   int32          `json:"audit_status"`                 // 审核状态
	IsActive      bool           `json:"is_active"`                    // 是否激活
}

type ProductInfoWithDistance struct {
	ProductInfo
	Distance float64 `json:"distance"` // 距离
}

type ProductInfoWithFavorite struct {
	ProductInfo
	IsFavorite bool `json:"is_favorite"` // 是否收藏
}

type ProductInfoWithFavoriteWithPagination struct {
	Points   []ProductInfoWithFavorite `json:"points"`
	Total    int64                     `json:"total"`
	Page     int64                     `json:"page"`
	PageSize int64                     `json:"page_size"`
}

// getProductList 获取产品列表
func getProductList(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取产品列表")

	userId := ec.AuthData.User.ID
	identityTag := int32(1)
	if userId > 0 {
		// 查询用户信息
		var user model.User
		err := ec.Nu.DB.Model(&model.User{}).Where("user_id = ?", userId).First(&user).Error
		if err != nil {
			sdlog.Errorf("获取用户信息失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
		identityTag = user.IdentityTag + 1
	}

	// 查询产品列表
	var products []model.Product
	err := ec.Nu.DB.Model(&model.Product{}).Where("level <= ? and category = ?", identityTag, "official").Find(&products).Error
	// err := ec.Nu.DB.Model(&model.Product{}).Where(" category = ?", "official").Find(&products).Error
	if err != nil {
		sdlog.Errorf("获取产品列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取所有产品ID
	productIDs := make([]int64, len(products))
	for i, p := range products {
		productIDs[i] = int64(p.ID)
	}

	// 查询产品图片
	var images []model.ProductImage
	err = ec.Nu.DB.Model(&model.ProductImage{}).
		Where("product_id IN ? AND status = ?", productIDs, 1).
		Order("product_id, sort_index").
		Find(&images).Error
	if err != nil {
		sdlog.Errorf("获取产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建产品ID到图片URL的映射
	productImages := make(map[int64][]ProductImage)
	for _, img := range images {
		productImages[img.ProductID] = append(productImages[img.ProductID], ProductImage{
			ID:        img.ID,
			ImageURL:  img.ImageURL,
			Status:    img.Status,
			ImageType: img.ImageType,
		})
	}

	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)
	// 转换响应数据
	productInfos := make([]ProductInfo, 0)
	for _, item := range products {
		productInfos = append(productInfos, ProductInfo{
			SpotID:        item.CheckInPointID,
			ProductID:     int64(item.ID),
			Name:          item.Name,
			Description:   item.Description,
			Amount:        decimal.NewFromFloat(item.Amount).InexactFloat64(),
			AmountUsd:     decimal.NewFromFloat(item.AmountUsd).InexactFloat64(),
			CheckCodeNum:  item.CheckCodeNum,
			HeadImg:       item.Image,
			Images:        productImages[int64(item.ID)],
			CommentCount:  item.CommentCount,
			VisitedCount:  item.VisitedCount,
			Tags:          productTagsToTagObjectList(strings.Split(item.Tags, ","), productTagMap),
			FavoriteCount: item.FavoriteCount,
			AuditStatus:   item.AuditStatus,
			IsActive:      item.IsActive,
		})
	}

	return webapi.OK(ProductListResponse{
		Products: productInfos,
	}).Render(ec)
}

func getProductListBySearch(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取产品列表")
	keyword := ec.QueryParams().Get("keyword")
	sortBy := ec.QueryParams().Get("sort_by") // 排序方式：distance(距离)、visited_count(浏览次数)，默认 无排序
	var products []model.Product
	err := ec.Nu.DB.Model(&model.Product{}).Where("name LIKE ?", "%"+keyword+"%").Where("audit_status = ?", 1).Find(&products).Limit(100).Error
	if err != nil {
		sdlog.Errorf("获取产品列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var spotIds []int64
	var productIds []int64
	for _, item := range products {
		productIds = append(productIds, int64(item.ID))
		spotIds = append(spotIds, item.CheckInPointID)
	}

	var productImages []model.ProductImage
	err = ec.Nu.DB.Model(&model.ProductImage{}).Where("product_id IN ?", productIds).Where("status = ?", common.SpotAuditStatusApproved).Find(&productImages).Error
	if err != nil {
		sdlog.Errorf("获取产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	productImagesMap := make(map[int64][]ProductImage)
	for _, img := range productImages {
		productImagesMap[img.ProductID] = append(productImagesMap[img.ProductID], ProductImage{
			ID:        img.ID,
			ImageURL:  img.ImageURL,
			Status:    img.Status,
			ImageType: img.ImageType,
		})
	}

	spotInfos, _, err := db.GetAllValidCheckPointsByIds(ec.Nu.DB, 0, 0, spotIds)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(ProductListResponse{
				Products: make([]ProductInfo, 0),
			}).Render(ec)
		}
		sdlog.Errorf("获取地标信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	sdlog.Error("spotInfos %v", spotInfos)
	spotInfosMap := make(map[int64]*db.CheckInPointWithOwnerByDistance)
	for _, item := range spotInfos {
		spotInfosMap[item.ID] = item
	}

	spotImagesMap, err := service.GetSpotImageMapBySpotIds(ec.Request().Context(), ec.Nu, spotIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		sdlog.Errorf("获取地标图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)
	productInfoWithDistance := make([]ProductInfoWithDistance, 0)
	for _, item := range products {
		spotInfo, ok := spotInfosMap[item.CheckInPointID]
		if !ok {
			continue
		}
		category, exists := memory.CategoryMemoryCache.GetCategoryList()[spotInfo.Category]
		if !exists {
			continue
		}
		categoryStr := ""
		if common.GetRequestLanguage() == common.LanguageZh {
			categoryStr = category.LabelZh
		} else {
			categoryStr = category.Label
		}
		spotImageurl := ""
		spotImage, ok := spotImagesMap[item.CheckInPointID]
		if ok {
			spotImageurl = spotImage[0].ImageURL
		}
		productInfoWithDistance = append(productInfoWithDistance, ProductInfoWithDistance{
			ProductInfo: ProductInfo{
				SpotID:            item.CheckInPointID,
				SpotTitleCn:       spotInfo.TitleCn,
				SpotTitleEn:       spotInfo.TitleEn,
				SpotDescriptionCn: spotInfo.DescriptionCn,
				SpotDescriptionEn: spotInfo.DescriptionEn,
				SpotImage:         spotImageurl,
				SpotCategory:      categoryStr,
				SpotCategoryEn:    category.Type,
				SpotCategoryImage: category.ImageURL,
				ProductID:         int64(item.ID),
				Name:              item.Name,
				Description:       item.Description,
				Amount:            decimal.NewFromFloat(item.Amount).InexactFloat64(),
				AmountUsd:         decimal.NewFromFloat(item.AmountUsd).InexactFloat64(),
				CheckCodeNum:      item.CheckCodeNum,
				HeadImg:           item.Image,
				Images:            productImagesMap[int64(item.ID)],
				CommentCount:      item.CommentCount,
				VisitedCount:      item.VisitedCount,
				Tags:              productTagsToTagObjectList(strings.Split(item.Tags, ","), productTagMap),
				FavoriteCount:     item.FavoriteCount,
				AuditStatus:       item.AuditStatus,
				IsActive:          item.IsActive,
				OwnerId:           spotInfo.OwnerID,
				Wechat:            spotInfo.Wechat,
			},
			Distance: spotInfo.Distance,
		})
	}
	if sortBy == "distance" {
		sort.Slice(productInfoWithDistance, func(i, j int) bool {
			return productInfoWithDistance[i].Distance < productInfoWithDistance[j].Distance
		})
	} else if sortBy == "favorite_count" {
		sort.Slice(productInfoWithDistance, func(i, j int) bool {
			return productInfoWithDistance[i].FavoriteCount > productInfoWithDistance[j].FavoriteCount
		})
	}

	return webapi.OK(ProductListResponseWithDistance{
		Products: productInfoWithDistance,
	}).Render(ec)
}

func getProductDetail(ec *middleware.AppRequestContext) error {
	productId := ec.QueryParams().Get("product_id")
	productIdInt, err := strconv.ParseInt(productId, 10, 64)
	if err != nil || productIdInt <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	isOwner := ec.QueryParams().Get("is_owner")
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	var product model.Product
	baseQuery := ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productIdInt)
	if isOwnerInt != 1 {
		baseQuery = baseQuery.Where("audit_status = ?", 1)
	}
	err = baseQuery.First(&product).Error
	if err != nil {
		sdlog.Errorf("获取产品详情失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	var productImages []model.ProductImage
	productImagesQuery := ec.Nu.DB.Model(&model.ProductImage{}).Where("product_id = ?", productId)
	if isOwnerInt != 1 {
		productImagesQuery = productImagesQuery.Where("status = ?", common.SpotAuditStatusApproved)
	}
	err = productImagesQuery.Find(&productImages).Error
	if err != nil {
		sdlog.Errorf("获取产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var spot model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", product.CheckInPointID).First(&spot).Error
	if err != nil {
		sdlog.Errorf("获取地标信息失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInPointNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var spotImage model.CheckInPointImage
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).Where("check_in_point_id = ?", product.CheckInPointID).Where("status = ?", 1).Order("sort_index").First(&spotImage).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("获取地标图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// save product visited count
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productId).Update("visited_count", gorm.Expr("visited_count + 1")).Error
	if err != nil {
		sdlog.Errorf("更新产品浏览次数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	service.AddUserProductVisitHistory(ec.Request().Context(), ec.Nu, ec.AuthData.User.ID, productIdInt)
	isFavorite := false
	userId := ec.AuthData.User.ID
	if userId > 0 {
		var count int64
		err = ec.Nu.DB.Model(&model.ProductFavorite{}).Where("user_id = ? AND product_id = ?", userId, productId).Count(&count).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				isFavorite = false
			} else {
				sdlog.Errorf("获取产品收藏次数失败: %v", err)
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			if count > 0 {
				isFavorite = true
			} else {
				isFavorite = false
			}
		}
	}
	headImg := ""
	images := make([]ProductImage, 0)
	for _, img := range productImages {
		images = append(images, ProductImage{
			ID:        img.ID,
			ImageURL:  img.ImageURL,
			Status:    img.Status,
			ImageType: img.ImageType,
		})
	}
	if len(images) > 0 {
		headImg = images[0].ImageURL
	}

	category, exists := memory.CategoryMemoryCache.GetCategoryList()[spot.Category]
	if !exists {
		return webapi.Error(common.ErrService).Render(ec)
	}
	categoryStr := ""
	if common.GetRequestLanguage() == common.LanguageZh {
		categoryStr = category.LabelZh
	} else {
		categoryStr = category.Label
	}
	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)
	productInfo := ProductInfoWithFavorite{
		ProductInfo: ProductInfo{
			SpotID:            product.CheckInPointID,
			SpotTitleCn:       spot.TitleCn,
			SpotTitleEn:       spot.TitleEn,
			SpotDescriptionCn: spot.DescriptionCn,
			SpotDescriptionEn: spot.DescriptionEn,
			SpotCategory:      categoryStr,
			SpotCategoryEn:    category.Type,
			SpotCategoryImage: category.ImageURL,
			SpotImage:         spotImage.ImageURL,
			OwnerId:           spot.OwnerID,
			ProductID:         int64(product.ID),
			Name:              product.Name,
			Description:       product.Description,
			Amount:            decimal.NewFromFloat(product.Amount).InexactFloat64(),
			AmountUsd:         decimal.NewFromFloat(product.AmountUsd).InexactFloat64(),
			CheckCodeNum:      product.CheckCodeNum,
			HeadImg:           headImg,
			Images:            images,
			CommentCount:      product.CommentCount,
			VisitedCount:      product.VisitedCount,
			Tags:              productTagsToTagObjectList(strings.Split(product.Tags, ","), productTagMap),
			FavoriteCount:     product.FavoriteCount,
			AuditStatus:       product.AuditStatus,
			IsActive:          product.IsActive,
			Wechat:            spot.Wechat,
			BelieveURL:        spot.BelieveURL,
			PcnURL:            spot.PcnURL,
			WhatsappURL:       spot.WhatsappURL,
			BelieveID:         spot.BelieveID,
			TelegramURL:       spot.TelegramURL,
		},
		IsFavorite: isFavorite,
	}
	return webapi.OK(productInfo).Render(ec)
}

type ProductListTagConfig struct {
	TagCn     string `json:"tag_cn"`
	TagEn     string `json:"tag_en"`
	SortIndex int32  `json:"sort_index"`
	Mark      string `json:"mark"`
}

func getProductListTagConfig(ec *middleware.AppRequestContext) error {
	tagConfig := make([]ProductListTagConfig, 0)
	err := ec.Nu.DB.Model(&model.ProductTagConfig{}).Find(&tagConfig).Order("sort_index").Error
	if err != nil {
		sdlog.Errorf("获取产品标签配置失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(tagConfig).Render(ec)
}

type ProductFavoriteParam struct {
	ProductId int64 `json:"product_id"`
}

func productFavoriteAdd(ec *middleware.AppRequestContext) error {
	req := ProductFavoriteParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	productId := req.ProductId
	userId := ec.AuthData.User.ID

	var product model.Product
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productId).First(&product).Error
	if err != nil {
		sdlog.Errorf("获取产品详情失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	err = ec.Nu.DB.Model(&model.ProductFavorite{}).Where("user_id = ? AND product_id = ?", userId, productId).First(&model.ProductFavorite{}).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrService).Render(ec)
		}
	} else {
		return webapi.OK(true).Render(ec)
	}
	product.FavoriteCount++
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productId).Update("favorite_count", product.FavoriteCount).Error
	if err != nil {
		sdlog.Errorf("更新产品收藏次数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	ec.Nu.DB.Model(&model.ProductFavorite{}).Create(&model.ProductFavorite{
		UserID:    userId,
		ProductID: int64(product.ID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	return webapi.OK(true).Render(ec)
}

func productFavoriteDelete(ec *middleware.AppRequestContext) error {
	req := ProductFavoriteParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userId := ec.AuthData.User.ID
	productId := req.ProductId
	var product model.Product
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productId).First(&product).Error
	if err != nil {
		sdlog.Errorf("获取产品详情失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	var productFavorite model.ProductFavorite
	err = ec.Nu.DB.Model(&model.ProductFavorite{}).Where("user_id = ? AND product_id = ?", userId, productId).First(&productFavorite).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductFavoriteNotExist).Render(ec)
		}
	}
	err = ec.Nu.DB.Model(&model.ProductFavorite{}).Where("user_id = ? AND product_id = ?", userId, productId).Delete(&model.ProductFavorite{}).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	product.FavoriteCount--
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", productId).Update("favorite_count", product.FavoriteCount).Error
	if err != nil {
		sdlog.Errorf("更新产品收藏次数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}

type ProductParam struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Chain    string  `json:"chain"`
}

type ProductCreateParam struct {
	SpotId      int64          `json:"spot_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ImageURLs   []string       `json:"image_urls"`
	Tags        []string       `json:"tags"`
	Payments    []ProductParam `json:"payments"`
}

type ProductUpdateParam struct {
	ProductId      int64          `json:"product_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	ImageURLs      []string       `json:"image_urls"`
	Tags           []string       `json:"tags"`
	Payments       []ProductParam `json:"payments"`
	DeleteImageIds []int64        `json:"delete_image_ids"`
}

func productCreate(ec *middleware.AppRequestContext) error {
	req := ProductCreateParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if len(req.Name) == 0 || utf8.RuneCountInString(req.Name) > 20 || len(req.Description) == 0 || utf8.RuneCountInString(req.Description) > 500 || len(req.Tags) == 0 || len(req.Payments) == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	var count int64
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ? and owner_id = ? and status = ?", req.SpotId, userId, common.SpotAuditStatusApproved).Count(&count).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInPointNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	if count == 0 {
		return webapi.Error(common.ErrCheckInPointNotExist).Render(ec)
	}

	amount := 0.0
	amountUsd := 0.0
	amountCurrency := ""
	hasValidPayment := false
	for _, payment := range req.Payments {
		if strings.ToLower(payment.Currency) == "fec" {
			amount = payment.Amount
			hasValidPayment = true
		} else if strings.ToLower(payment.Currency) == "usdt" {
			amountUsd = payment.Amount
			hasValidPayment = true
			amountCurrency = "USDT"
		}
	}
	if !hasValidPayment {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	product := model.Product{
		CheckInPointID: req.SpotId,
		Name:           req.Name,
		Description:    req.Description,
		Tags:           strings.Join(req.Tags, ","),
		Amount:         decimal.NewFromFloat(amount).InexactFloat64(),
		AmountUsd:      decimal.NewFromFloat(amountUsd).InexactFloat64(),
		CheckCodeNum:   0,
		Level:          1,
		LevelTag:       "",
		Category:       "unofficial",
		AuditStatus:    0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Image:          fmt.Sprintf("%s/%s", ec.Nu.Config.Aws.AwsS3CustomDomain, req.ImageURLs[0]),
		OriginalPrice:  0,
		AmountCurrency: amountCurrency,
		VisitedCount:   0,
		CommentCount:   0,
		FavoriteCount:  0,
		IsActive:       true,
	}

	err = ec.Nu.DB.Model(&model.Product{}).Create(&product).Error
	if err != nil {
		sdlog.Errorf("创建产品失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	err = service.CreateProductImages(ec.Request().Context(), ec.Nu, ec.Nu.DB, userId, req.ImageURLs, int64(product.ID))
	if err != nil {
		sdlog.Errorf("创建产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

func productUpdate(ec *middleware.AppRequestContext) error {
	req := ProductUpdateParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	ok, err := service.CheckUserIsProductOwner(ec.Request().Context(), ec.Nu, userId, req.ProductId)
	if err != nil {
		sdlog.Errorf("检查用户是否是商品所有者失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if !ok {
		return webapi.Error(common.ErrNotFoundProduct).Render(ec)
	}

	product := model.Product{}
	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", req.ProductId).First(&product).Error
	if err != nil {
		return webapi.Error(common.ErrNotFoundProduct).Render(ec)
	}

	amount := 0.0
	amountUsd := 0.0
	amountCurrency := ""
	hasValidPayment := false
	for _, payment := range req.Payments {
		if strings.ToLower(payment.Currency) == "fec" {
			amount = payment.Amount
			hasValidPayment = true
		} else if strings.ToLower(payment.Currency) == "usdt" {
			amountUsd = payment.Amount
			hasValidPayment = true
			amountCurrency = "USDT"
		}
	}
	if !hasValidPayment {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Tags = strings.Join(req.Tags, ",")
	product.Amount = decimal.NewFromFloat(amount).InexactFloat64()
	product.AmountUsd = decimal.NewFromFloat(amountUsd).InexactFloat64()
	product.AmountCurrency = amountCurrency
	product.UpdatedAt = time.Now()
	product.AuditStatus = common.SpotAuditStatusPending

	// 先删除旧的图片记录
	if len(req.DeleteImageIds) > 0 {
		if err = ec.Nu.DB.Where("product_id = ? and id in (?)", req.ProductId, req.DeleteImageIds).Delete(&model.ProductImage{}).Error; err != nil {
			sdlog.Errorf("[productUpdate] 删除旧图片记录失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	if len(req.ImageURLs) > 0 {
		// 添加图片
		err = service.CreateProductImages(ec.Request().Context(), ec.Nu, ec.Nu.DB, userId, req.ImageURLs, int64(product.ID))
		if err != nil {
			sdlog.Errorf("创建产品图片失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", req.ProductId).Updates(product).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

type ProductWithSpotInfoList struct {
	Points   []CheckInPointInfoStructV1 `json:"points"`
	Total    int64                      `json:"total"`
	Page     int64                      `json:"page"`
	PageSize int64                      `json:"page_size"`
}

type CheckInPointInfoStructV1 struct {
	ID            int64                        `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	OwnerId       int64                        `gorm:"column:owner_id;not null" json:"owner_id"`
	Wechat        string                       `gorm:"column:wechat;not null" json:"wechat"`
	Title         string                       `gorm:"column:title;not null" json:"title"`
	Category      string                       `gorm:"column:category;not null" json:"category"`
	CategoryImage string                       `gorm:"column:category_image" json:"category_image"`
	CategoryEn    string                       `gorm:"column:category_en" json:"category_en"`
	Currency      string                       `gorm:"column:currency;not null" json:"currency"`
	TelegramURL   string                       `gorm:"column:telegram_url;not null" json:"telegram_url"`
	DescriptionCn string                       `gorm:"column:description_cn" json:"description_cn"`
	DescriptionEn string                       `gorm:"column:description_en" json:"description_en"`
	TitleCn       string                       `gorm:"column:title_cn;not null" json:"title_cn"`
	TitleEn       string                       `gorm:"column:title_en;not null" json:"title_en"`
	LikeCount     int64                        `gorm:"column:like_count;not null" json:"like_count"`
	CommentCount  int64                        `gorm:"column:comment_count;not null" json:"comment_count"`
	Images        []string                     `json:"images"`
	Products      []ProductInfoWithOutSpotInfo `json:"products"`
}

type SpotWithProduct struct {
	model.CheckInPoint
	Products []model.Product `gorm:"foreignKey:CheckInPointID" json:"products"`
}

// 查询地标下的产品，按照地标创建时间排序
// 先查询所有对应分页商品数量大于0的地标，需要连表join查询，然后查询地标下的产品，按照地标创建时间排序，每页20条
func getUserProduct(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	unpublished := ec.QueryParams().Get("unpublished")
	unpublishedBool := sdparse.BoolDef(unpublished, false)
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	var total int64
	// 先统计总数（只统计有产品的 check_in_point）
	baseQuery := ec.Nu.DB.Model(&model.CheckInPoint{}).
		Joins("JOIN product p ON p.check_in_point_id = check_in_point.id").
		Where("check_in_point.owner_id = ?", userId)
	if unpublishedBool {
		baseQuery = baseQuery.Where("p.is_active = ?", false)
	} else {
		baseQuery = baseQuery.Where("p.is_active = ?", true)
	}
	err := baseQuery.
		Distinct("check_in_point.id").
		Count(&total).Error
	if err != nil {
		return err
	}

	var result []SpotWithProduct
	// 分页查询数据 + 预加载 Products
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("check_in_point.owner_id = ? AND EXISTS (SELECT 1 FROM product WHERE product.check_in_point_id = check_in_point.id AND product.is_active = ?)", userId, !unpublishedBool).
		Order("check_in_point.created_at DESC").
		Preload("Products").
		Offset((int(pageInt) - 1) * int(pageSizeInt)).
		Limit(int(pageSizeInt)).
		Find(&result).Error
	if err != nil {
		return err
	}

	spotIds := make([]int64, len(result))
	productIds := make([]int64, 0)
	for i, spot := range result {
		spotIds[i] = spot.ID
		for _, product := range spot.Products {
			productIds = append(productIds, int64(product.ID))
		}
	}

	// 查询所有check_in_points 第一张图片
	var checkInPointImages []model.CheckInPointImage
	subQuery := ec.Nu.DB.
		Table("check_in_point_image AS ci2").
		Select("MIN(sort_index)").
		Where("ci2.check_in_point_id = ci.check_in_point_id AND ci2.status = ?", common.SpotAuditStatusApproved)

	err = ec.Nu.DB.Table("check_in_point_image AS ci").
		Where("ci.check_in_point_id IN (?)", spotIds).
		Where("ci.sort_index = (?)", subQuery).
		Find(&checkInPointImages).Error

	checkInPointImageMap := make(map[int64]string)
	for _, image := range checkInPointImages {
		checkInPointImageMap[image.CheckInPointID] = image.ImageURL
	}

	// 查询所有 第一张图片
	var productImages []model.ProductImage
	subQuery = ec.Nu.DB.
		Table("product_image AS pi2").
		Select("MIN(sort_index)").
		Where("pi2.product_id = pi.product_id AND pi2.status = ?", common.SpotAuditStatusApproved)

	err = ec.Nu.DB.Table("product_image AS pi").
		Where("pi.product_id IN (?)", productIds).
		Where("pi.sort_index = (?)", subQuery).
		Find(&productImages).Error

	productImageMap := make(map[int64]ProductImage)
	for _, image := range productImages {
		productImageMap[image.ProductID] = ProductImage{
			ID:        image.ID,
			ImageURL:  image.ImageURL,
			Status:    image.Status,
			ImageType: image.ImageType,
		}
	}
	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)

	var points []CheckInPointInfoStructV1
	for _, spot := range result {
		spotImages := make([]string, 0)
		if image, ok := checkInPointImageMap[spot.ID]; ok {
			spotImages = append(spotImages, image)
		}
		category, exists := memory.CategoryMemoryCache.GetCategoryList()[spot.Category]
		if !exists {
			continue
		}
		categoryStr := ""
		if common.GetRequestLanguage() == common.LanguageZh {
			categoryStr = category.LabelZh
		} else {
			categoryStr = category.Label
		}
		point := CheckInPointInfoStructV1{
			ID:            spot.ID,
			OwnerId:       spot.OwnerID,
			Wechat:        spot.Wechat,
			Title:         spot.Title,
			Category:      categoryStr,
			CategoryImage: category.ImageURL,
			CategoryEn:    category.Type,
			Currency:      spot.Currency,
			TelegramURL:   spot.TelegramURL,
			DescriptionCn: spot.DescriptionCn,
			DescriptionEn: spot.DescriptionEn,
			TitleCn:       spot.TitleCn,
			TitleEn:       spot.TitleEn,
			LikeCount:     spot.LikeCount,
			CommentCount:  spot.CommentCount,
			Images:        spotImages,
			Products:      make([]ProductInfoWithOutSpotInfo, 0),
		}
		for _, product := range spot.Products {
			productImages := make([]ProductImage, 0)
			if image, ok := productImageMap[int64(product.ID)]; ok {
				productImages = append(productImages, ProductImage{
					ID:        image.ID,
					ImageURL:  image.ImageURL,
					Status:    image.Status,
					ImageType: image.ImageType,
				})
			}
			headImg := ""
			if len(productImages) > 0 {
				headImg = productImages[0].ImageURL
			}
			productInfo := ProductInfoWithOutSpotInfo{
				ProductID:     int64(product.ID),
				Name:          product.Name,
				Description:   product.Description,
				Amount:        decimal.NewFromFloat(product.Amount).InexactFloat64(),
				AmountUsd:     decimal.NewFromFloat(product.AmountUsd).InexactFloat64(),
				CheckCodeNum:  product.CheckCodeNum,
				HeadImg:       headImg,
				Images:        productImages,
				CommentCount:  product.CommentCount,
				VisitedCount:  product.VisitedCount,
				Tags:          productTagsToTagObjectList(strings.Split(product.Tags, ","), productTagMap),
				FavoriteCount: product.FavoriteCount,
				AuditStatus:   product.AuditStatus,
				IsActive:      product.IsActive,
			}
			addProduct := false
			if unpublishedBool {
				if product.IsActive == false {
					addProduct = true
				}
			} else {
				if product.IsActive == true {
					addProduct = true
				}
			}
			if addProduct {
				point.Products = append(point.Products, productInfo)
			}
		}
		points = append(points, point)
	}

	return webapi.OK(ProductWithSpotInfoList{
		Points:   points,
		Total:    total,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}).Render(ec)
}

func getUserProductFavorites(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	total, err := service.GetUserFavoritesProductCount(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		sdlog.Errorf("获取用户产品收藏数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	productFavorites, err := service.GetUserFavoritesProduct(ec.Request().Context(), ec.Nu, userId, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("获取用户产品收藏失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	productIds := make([]int64, len(productFavorites))
	spotIds := make([]int64, len(productFavorites))
	for i, product := range productFavorites {
		productIds[i] = int64(product.ID)
		spotIds[i] = product.CheckInPointID
	}

	var spots []model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id IN (?)", spotIds).Find(&spots).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var spotImages []model.CheckInPointImage
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).Where("check_in_point_id IN (?)", spotIds).Where("status = ?", 1).Order("sort_index").Find(&spotImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	spotImageMap := make(map[int64]model.CheckInPointImage)
	for _, spotImage := range spotImages {
		spotImageMap[spotImage.CheckInPointID] = spotImage
	}

	spotMap := make(map[int64]model.CheckInPoint)
	for _, spot := range spots {
		spotMap[spot.ID] = spot
	}

	// 查询所有商品 第一张图片
	var productImages []model.ProductImage
	subQuery := ec.Nu.DB.
		Table("product_image AS pi2").
		Select("MIN(sort_index)").
		Where("pi2.product_id = pi.product_id AND pi2.status = ?", common.SpotAuditStatusApproved)

	err = ec.Nu.DB.Table("product_image AS pi").
		Where("pi.product_id IN (?)", productIds).
		Where("pi.sort_index = (?)", subQuery).
		Find(&productImages).Error

	productImageMap := make(map[int64]ProductImage)
	for _, image := range productImages {
		productImageMap[image.ProductID] = ProductImage{
			ID:        image.ID,
			ImageURL:  image.ImageURL,
			Status:    image.Status,
			ImageType: image.ImageType,
		}
	}

	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)
	productFavoritesNew := make([]ProductInfoWithFavorite, 0)
	for _, product := range productFavorites {
		imageUrl := ""
		if image, ok := productImageMap[int64(product.ID)]; ok {
			imageUrl = image.ImageURL
		}

		spotImageUrl := ""
		if spotImage, ok := spotImageMap[product.CheckInPointID]; ok {
			spotImageUrl = spotImage.ImageURL
		}

		spot, ok := spotMap[product.CheckInPointID]
		if !ok {
			continue
		}

		productInfo := ProductInfoWithFavorite{
			ProductInfo: ProductInfo{
				SpotID:            product.CheckInPointID,
				SpotTitleCn:       spot.TitleCn,
				SpotTitleEn:       spot.TitleEn,
				SpotDescriptionCn: spot.DescriptionCn,
				SpotDescriptionEn: spot.DescriptionEn,
				SpotImage:         spotImageUrl,
				OwnerId:           spot.OwnerID,
				ProductID:         int64(product.ID),
				Name:              product.Name,
				Description:       product.Description,
				Amount:            decimal.NewFromFloat(product.Amount).InexactFloat64(),
				AmountUsd:         decimal.NewFromFloat(product.AmountUsd).InexactFloat64(),
				CheckCodeNum:      product.CheckCodeNum,
				HeadImg:           imageUrl,
				Images:            make([]ProductImage, 0),
				CommentCount:      product.CommentCount,
				VisitedCount:      product.VisitedCount,
				Tags:              productTagsToTagObjectList(strings.Split(product.Tags, ","), productTagMap),
				FavoriteCount:     product.FavoriteCount,
				AuditStatus:       product.AuditStatus,
				IsActive:          product.IsActive,
				Wechat:            spot.Wechat,
				BelieveURL:        spot.BelieveURL,
				PcnURL:            spot.PcnURL,
				WhatsappURL:       spot.WhatsappURL,
				BelieveID:         spot.BelieveID,
				TelegramURL:       spot.TelegramURL,
			},
			IsFavorite: true,
		}
		productFavoritesNew = append(productFavoritesNew, productInfo)
	}

	return webapi.OK(ProductInfoWithFavoriteWithPagination{
		Points:   productFavoritesNew,
		Total:    total,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}).Render(ec)
}

func getUserProductVisitHistory(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	total, err := service.GetUserProductVisitHistoryCount(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		sdlog.Errorf("获取用户产品浏览历史数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	products, err := service.GetUserProductVisitHistory(ec.Request().Context(), ec.Nu, userId, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("获取用户产品浏览历史失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	productIds := make([]int64, len(products))
	spotIds := make([]int64, len(products))
	for i, product := range products {
		spotIds[i] = product.CheckInPointID
		productIds = append(productIds, int64(product.ID))
	}

	var spots []model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id IN (?)", spotIds).Find(&spots).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	spotMap := make(map[int64]model.CheckInPoint)
	for _, spot := range spots {
		spotMap[spot.ID] = spot
	}

	var spotImages []model.CheckInPointImage
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).Where("check_in_point_id IN (?)", spotIds).Where("status = ?", common.SpotAuditStatusApproved).Order("sort_index").Find(&spotImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	spotImageMap := make(map[int64]model.CheckInPointImage)
	for _, spotImage := range spotImages {
		spotImageMap[spotImage.CheckInPointID] = spotImage
	}

	productImageMap, err := service.GetProductImageMapByProductIds(ec.Request().Context(), ec.Nu, productIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	productTagMap, _ := service.GetProductTagMap(ec.Request().Context(), ec.Nu)
	productVisitHistoryNew := make([]ProductInfoWithFavorite, 0)
	for _, product := range products {
		spotImageUrl := ""
		if spotImage, ok := spotImageMap[product.CheckInPointID]; ok {
			spotImageUrl = spotImage.ImageURL
		}

		productImages := productImageMap[int64(product.ID)]
		headImg := ""
		if len(productImages) > 0 {
			headImg = productImages[0].ImageURL
		}

		productImagesNew := make([]ProductImage, 0)
		for _, productImage := range productImages {
			productImagesNew = append(productImagesNew, ProductImage{
				ID:        productImage.ID,
				ImageURL:  productImage.ImageURL,
				Status:    productImage.Status,
				ImageType: productImage.ImageType,
			})
		}

		spot, ok := spotMap[product.CheckInPointID]
		if !ok {
			continue
		}

		productInfo := ProductInfoWithFavorite{
			ProductInfo: ProductInfo{
				SpotID:            product.CheckInPointID,
				SpotTitleCn:       spot.TitleCn,
				SpotTitleEn:       spot.TitleEn,
				SpotDescriptionCn: spot.DescriptionCn,
				SpotDescriptionEn: spot.DescriptionEn,
				SpotImage:         spotImageUrl,
				OwnerId:           spot.OwnerID,
				ProductID:         int64(product.ID),
				Name:              product.Name,
				Description:       product.Description,
				Amount:            decimal.NewFromFloat(product.Amount).InexactFloat64(),
				AmountUsd:         decimal.NewFromFloat(product.AmountUsd).InexactFloat64(),
				CheckCodeNum:      product.CheckCodeNum,
				HeadImg:           headImg,
				Images:            productImagesNew,
				CommentCount:      product.CommentCount,
				VisitedCount:      product.VisitedCount,
				Tags:              productTagsToTagObjectList(strings.Split(product.Tags, ","), productTagMap),
				FavoriteCount:     product.FavoriteCount,
				AuditStatus:       product.AuditStatus,
				IsActive:          product.IsActive,
				Wechat:            spot.Wechat,
				BelieveURL:        spot.BelieveURL,
				PcnURL:            spot.PcnURL,
				WhatsappURL:       spot.WhatsappURL,
				BelieveID:         spot.BelieveID,
				TelegramURL:       spot.TelegramURL,
			},
			IsFavorite: false,
		}
		productVisitHistoryNew = append(productVisitHistoryNew, productInfo)
	}

	return webapi.OK(ProductInfoWithFavoriteWithPagination{
		Points:   productVisitHistoryNew,
		Total:    total,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}).Render(ec)
}

type ProductPublishParam struct {
	ProductId int64 `json:"product_id"`
}

func productPublish(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	param := ProductPublishParam{}
	err := ec.Bind(&param)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	ok, err := service.CheckUserIsProductOwner(ec.Request().Context(), ec.Nu, userId, param.ProductId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if !ok {
		return webapi.Error(common.ErrNotFoundProduct).Render(ec)
	}

	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", param.ProductId).Update("is_active", true).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

func productUnpublish(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	param := ProductPublishParam{}
	err := ec.Bind(&param)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	ok, err := service.CheckUserIsProductOwner(ec.Request().Context(), ec.Nu, userId, param.ProductId)
	if err != nil {
		sdlog.Errorf("检查用户是否是商品所有者失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if !ok {
		return webapi.Error(common.ErrNotFoundProduct).Render(ec)
	}

	err = ec.Nu.DB.Model(&model.Product{}).Where("id = ?", param.ProductId).Update("is_active", false).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

func productTagsToTagObjectList(tags []string, productTagMap map[string]model.ProductTagConfig) []ProductTag {

	tagObjectList := make([]ProductTag, 0)
	for _, tag := range tags {
		if productTag, ok := productTagMap[tag]; ok {
			tagObjectList = append(tagObjectList, ProductTag{
				TagCn:     productTag.TagCn,
				TagEn:     productTag.TagEn,
				SortIndex: productTag.SortIndex,
				Mark:      tag,
			})
		}
	}
	return tagObjectList
}

// CreateOrderRequestV2 创建订单请求参数
type CreateProductOrderRequest struct {
	ProductID      int64  `json:"product_id"`      // 产品ID
	PayType        string `json:"pay_type"`        // 支付方式
	Currency       string `json:"currency"`        // 货币
	Num            int32  `json:"num"`             // 数量
	ContactName    string `json:"contact_name"`    // 联系人姓名
	ContactPhone   string `json:"contact_phone"`   // 联系人电话
	ContactAddress string `json:"contact_address"` // 联系人地址
}

// CreateOrderResponseV2 创建订单响应
type CreateProductOrderResponse struct {
	OrderID      int64     `json:"order_id"`      // 订单ID
	ProductID    int64     `json:"product_id"`    // 产品ID
	ProductName  string    `json:"product_name"`  // 产品名称
	ProductNum   int32     `json:"product_num"`   // 产品数量
	ProductPrice float64   `json:"product_price"` // 产品价格
	Amount       float64   `json:"amount"`        // 订单金额
	PayType      string    `json:"pay_type"`      // 支付方式
	Currency     string    `json:"currency"`      // 货币
	CreatedAt    time.Time `json:"created_at"`
	ExpiredAt    time.Time `json:"expired_at"` // 订单过期时间
	IsProduct    bool      `json:"is_product"` // 是否是产品
}

// createProductOrder 创建产品订单
func createProductOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始创建产品订单")

	// 解析请求参数
	req := CreateProductOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 参数验证
	if req.ProductID <= 0 {
		sdlog.Error("产品ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Num <= 0 || req.ContactName == "" || req.ContactPhone == "" || req.ContactAddress == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询支付方式
	var payType model.AddressCheckpoint
	err := ec.Nu.DB.Model(&model.AddressCheckpoint{}).
		Where("currency = ?", req.Currency).
		Where("chain = ?", req.PayType).
		Where("is_active = ?", true).
		First(&payType).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("查询支付方式失败: %v", err)
			return webapi.Error(common.ErrParam).Render(ec)
		}
		sdlog.Errorf("查询支付方式失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询产品信息
	var product model.Product
	err = ec.Nu.DB.Model(&model.Product{}).
		Where("id = ?", req.ProductID).
		Where("is_active = ?", true).
		Where("audit_status = ?", common.SpotAuditStatusApproved).
		First(&product).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("产品不存在: %d", req.ProductID)
			return webapi.Error(common.ErrNotFoundProduct).Render(ec)
		}
		sdlog.Errorf("查询产品信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if req.Currency == "FEC" && product.Amount == 0 {
		return webapi.Error(common.ErrProductNotSupportThisCurrency).Render(ec)
	}

	if req.Currency == "USDT" && product.AmountUsd == 0 {
		return webapi.Error(common.ErrProductNotSupportThisCurrency).Render(ec)
	}

	spot := model.CheckInPoint{}
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id = ?", product.CheckInPointID).
		First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrNotFoundProduct).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建订单
	productAmount := product.Amount * float64(req.Num)
	productPrice := product.Amount
	if req.PayType != "FEC" {
		productAmount = product.AmountUsd * float64(req.Num)
		productPrice = product.AmountUsd
	}
	orderID, _ := strconv.ParseInt(sdrand.GenerateSecureString(10, true, false, false), 10, 64)
	order := &model.ProductOrder{
		OrderID:        orderID,
		UserID:         ec.AuthData.User.ID,
		SellerID:       int64(spot.OwnerID),
		ProductID:      int64(product.ID),
		SpotID:         int64(product.CheckInPointID),
		ProductName:    product.Name,
		ProductNum:     req.Num,
		ProductPrice:   decimal.NewFromFloat(productPrice).InexactFloat64(),
		Amount:         decimal.NewFromFloat(productAmount).InexactFloat64(),
		PayType:        req.PayType,
		Unit:           req.Currency,
		Status:         common.ProductOrderStatusPendingPayment, // 0: 待支付
		RefundStatus:   common.ProductRefundStatusNoRefund,
		ContactName:    req.ContactName,
		ContactPhone:   req.ContactPhone,
		ContactAddress: req.ContactAddress,
		IsCommented:    false,
		CancelByBuyer:  false,
		CancelReason:   "",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = ec.Nu.DB.Create(order).Error
	if err != nil {
		sdlog.Errorf("创建订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("创建订单成功，订单ID: %d", order.ID)
	return webapi.OK(CreateProductOrderResponse{
		OrderID:      order.OrderID,
		ProductID:    order.ProductID,
		ProductName:  order.ProductName,
		Amount:       decimal.NewFromFloat(order.Amount).InexactFloat64(),
		ProductNum:   order.ProductNum,
		ProductPrice: decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
		PayType:      order.PayType,
		Currency:     order.Unit,
		CreatedAt:    order.CreatedAt.Truncate(time.Second),
		ExpiredAt:    order.CreatedAt.Truncate(time.Second).Add(15 * time.Minute),
		IsProduct:    true,
	}).Render(ec)
}

type ContactInfo struct {
	ContactName    string `json:"contact_name"`    // 联系人姓名
	ContactPhone   string `json:"contact_phone"`   // 联系人电话
	ContactAddress string `json:"contact_address"` // 联系人地址
}

type RefundInfo struct {
	RefundID  int64     `json:"refund_id"`  // 退款ID
	Amount    float64   `json:"amount"`     // 退款金额
	Unit      string    `json:"unit"`       // 货币
	Status    string    `json:"status"`     // 退款状态
	Remark    string    `json:"remark"`     // 退款备注
	CreatedAt time.Time `json:"created_at"` // 退款创建时间
}

// OrderListItem 订单列表项
type ProductOrderDetail struct {
	OrderID      int64     `json:"order_id"`      // 订单ID
	SpotID       int64     `json:"spot_id"`       // 景点ID
	SpotTitleCn  string    `json:"spot_title_cn"` // 景点名称
	SpotTitleEn  string    `json:"spot_title_en"` // 景点名称
	ProductID    int64     `json:"product_id"`    // 产品ID
	SellerID     int64     `json:"seller_id"`     // 卖家ID
	ProductName  string    `json:"product_name"`  // 产品名称
	ProductNum   int32     `json:"product_num"`   // 产品数量
	ProductPrice float64   `json:"product_price"` // 产品价格
	Amount       float64   `json:"amount"`        // 订单金额
	Status       string    `json:"status"`        // 订单状态
	PayType      string    `json:"pay_type"`      // 支付方式
	Unit         string    `json:"unit"`          // 货币
	CreatedAt    string    `json:"created_at"`    // 创建时间
	CAt          time.Time `json:"c_at"`          // 创建时间v2
	UAt          time.Time `json:"u_at"`          // 更新时间v2
	ExpiredAt    time.Time `json:"expired_at"`    // 订单过期时间
	UpdatedAt    string    `json:"updated_at"`    // 更新时间
	IsProduct    bool      `json:"is_product"`    // 是否是产品

	ContactInfo   ContactInfo `json:"contact_info"`    // 联系人信息
	RefundInfo    RefundInfo  `json:"refund_info"`     // 退款信息
	IsCommented   bool        `json:"is_commented"`    // 是否已评论
	RefundStatus  string      `json:"refund_status"`   // 退款状态
	CancelByBuyer bool        `json:"cancel_by_buyer"` // 是否已取消收货
	CancelReason  string      `json:"cancel_reason"`   // 取消原因

	// 联系方式
	Wechat      string `json:"wechat"`       // 联系方式
	BelieveURL  string `json:"believe_url"`  // 联系方式
	PcnURL      string `json:"pcn_url"`      // 联系方式
	WhatsappURL string `json:"whatsapp_url"` // 联系方式
	BelieveID   string `json:"believe_id"`   // 联系方式
	TelegramURL string `json:"telegram_url"` // 联系方式
	// 产品详细信息
	Product struct {
		ID             int32    `json:"id"`              // 产品ID
		Name           string   `json:"name"`            // 产品名称
		Description    string   `json:"description"`     // 产品描述
		Amount         float64  `json:"amount"`          // 产品价格
		CheckCodeNum   int32    `json:"check_code_num"`  // 检查码数量
		Category       string   `json:"category"`        // 产品分类
		Image          string   `json:"image"`           // 产品头图
		AmountCurrency string   `json:"amount_currency"` // 货币类型
		AmountUsd      float64  `json:"amount_usd"`      // USD价格
		Images         []string `json:"images"`          // 产品图片列表
	} `json:"product"`
}

func getProductOrderDetail(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	orderId := ec.QueryParam("order_id")
	isSeller := sdparse.BoolDef(ec.QueryParam("is_seller"), false)
	if orderId == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderIdInt, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil || orderIdInt <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	if isSeller {
		err = ec.Nu.DB.Model(&model.ProductOrder{}).
			Where("order_id = ?", orderIdInt).
			Where("seller_id = ?", userId).
			First(&order).Error
	} else {
		err = ec.Nu.DB.Model(&model.ProductOrder{}).
			Where("order_id = ?", orderIdInt).
			Where("user_id = ?", userId).
			First(&order).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductOrderNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	spot := model.CheckInPoint{}
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id = ?", order.SpotID).
		First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrNotFoundProduct).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	product := model.Product{}
	err = ec.Nu.DB.Model(&model.Product{}).
		Where("id = ?", order.ProductID).
		First(&product).Error
	if err != nil {
		return webapi.Error(common.ErrNotFoundProduct).Render(ec)
	}

	// 查询产品图片
	productImages := make([]model.ProductImage, 0)
	err = ec.Nu.DB.Model(&model.ProductImage{}).
		Where("product_id = ?", order.ProductID).
		Where("status = ?", common.SpotAuditStatusApproved).
		Find(&productImages).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	productImagesNew := make([]string, 0)
	for _, productImage := range productImages {
		productImagesNew = append(productImagesNew, productImage.ImageURL)
	}

	headImg := ""
	if len(productImagesNew) > 0 {
		headImg = productImagesNew[0]
	}

	// 查询退款信息
	refundInfo := model.ProductRefund{}
	err = ec.Nu.DB.Model(&model.ProductRefund{}).
		Where("order_id = ?", order.OrderID).
		Where("user_id = ?", order.UserID).
		Order("created_at DESC").
		First(&refundInfo).Error
	if err != nil {
		sdlog.Errorf("查询退款信息失败: %v", err)
	}
	refundInfoNew := RefundInfo{}
	if refundInfo.ID > 0 {
		refundInfoNew = RefundInfo{
			RefundID:  refundInfo.ID,
			Amount:    decimal.NewFromFloat(refundInfo.Amount).InexactFloat64(),
			Unit:      refundInfo.Unit,
			Status:    refundInfo.Status,
			Remark:    refundInfo.Remark,
			CreatedAt: refundInfo.CreatedAt,
		}
		sdlog.Infof("查询退款信息成功: %v", refundInfoNew)
	} else {
		sdlog.Errorf("查询退款信息失败: 不存在退款")
	}

	amount := decimal.NewFromFloat(order.Amount)
	productDetail := ProductOrderDetail{
		OrderID:       order.OrderID,
		SpotID:        order.SpotID,
		SpotTitleCn:   spot.TitleCn,
		SpotTitleEn:   spot.TitleEn,
		ProductID:     order.ProductID,
		SellerID:      order.SellerID,
		ProductName:   product.Name,
		ProductNum:    order.ProductNum,
		ProductPrice:  decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
		Amount:        amount.InexactFloat64(),
		Status:        order.Status,
		PayType:       order.PayType,
		Unit:          order.Unit,
		CreatedAt:     order.CreatedAt.Format("2006-01-02 15:04:05"),
		CAt:           order.CreatedAt,
		UAt:           order.UpdatedAt,
		ExpiredAt:     order.CreatedAt.Add(15 * time.Minute),
		UpdatedAt:     order.UpdatedAt.Format("2006-01-02 15:04:05"),
		IsProduct:     true,
		RefundStatus:  order.RefundStatus,
		CancelByBuyer: order.CancelByBuyer,
		CancelReason:  order.CancelReason,
		IsCommented:   order.IsCommented,
	}

	productDetail.ContactInfo = ContactInfo{
		ContactName:    order.ContactName,
		ContactPhone:   order.ContactPhone,
		ContactAddress: order.ContactAddress,
	}

	productDetail.RefundInfo = refundInfoNew
	productDetail.Product.Category = product.Category
	productDetail.Product.Image = headImg
	productDetail.Product.Images = productImagesNew
	productDetail.Product.ID = int32(product.ID)
	productDetail.Product.Name = product.Name
	productDetail.Product.Description = product.Description
	productDetail.Product.Amount = decimal.NewFromFloat(product.Amount).InexactFloat64()
	productDetail.Product.AmountCurrency = product.AmountCurrency
	productDetail.Product.AmountUsd = product.AmountUsd
	productDetail.Wechat = spot.Wechat
	productDetail.BelieveURL = spot.BelieveURL
	productDetail.PcnURL = spot.PcnURL
	productDetail.WhatsappURL = spot.WhatsappURL
	productDetail.BelieveID = spot.BelieveID
	productDetail.TelegramURL = spot.TelegramURL

	return webapi.OK(productDetail).Render(ec)
}

// getOrderPayStatus 获取订单支付状态
func getProductOrderPayStatus(ec *middleware.AppRequestContext) error {
	orderIDStr := ec.QueryParams().Get("order_id")
	if orderIDStr == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		sdlog.Error("订单ID格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var order model.ProductOrder
	err = ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %d", orderID)
			return webapi.Error(common.ErrOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	resp := GetOrderPayStatusResponse{
		OrderID: order.OrderID,
		Status:  order.Status,
	}
	return webapi.OK(resp).Render(ec)
}

type CancelProductOrderRequest struct {
	OrderID int64  `json:"order_id"` // 订单ID
	Reason  string `json:"reason"`   // 取消原因
}

// CancelOrder 取消订单
func cancelProductOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始取消订单")

	// 获取订单ID
	req := CancelProductOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 || req.Reason == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单
	var order model.ProductOrder
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查订单状态是否可以取消
	if order.Status != common.ProductOrderStatusPendingPayment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 更新订单状态为已取消
	err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusCancelled).
		Update("cancel_reason", req.Reason).
		Update("cancel_by_buyer", true).
		Update("cancel_at", time.Now().Unix()).
		Update("refund_status", common.ProductRefundStatusNoRefund).
		Update("updated_at", time.Now()).Error
	if err != nil {
		sdlog.Errorf("取消订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 若存在支付订单，则取消支付订单
	var payOrder model.PaymentOrder
	err = ec.Nu.DB.Model(&model.PaymentOrder{}).
		Where("order_id = ?", orderID).
		First(&payOrder).Error
	if err != nil {
		sdlog.Errorf("查询支付订单失败: %v", err)
	}
	if payOrder.Status == common.OrderStatusUnpaid {
		err = ec.Nu.DB.Model(&payOrder).Update("status", common.OrderStatusCancel).Error
		if err != nil {
			sdlog.Errorf("取消支付订单失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// 删除对应的退款订单
	err = ec.Nu.DB.Model(&model.ProductRefund{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		Delete(&model.ProductRefund{}).Error
	if err != nil {
		sdlog.Errorf("删除退款订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	sdlog.Infof("订单 %s 取消成功", orderID)
	return webapi.OK(true).Render(ec)
}

// 商户取消订单
func cancelProductOrderByMerchant(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始取消订单")

	req := CancelProductOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 || req.Reason == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingPayment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 查询商户是不是属于该用户
	var spot model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id = ?", order.SpotID).
		Where("owner_id = ?", ec.AuthData.User.ID).
		First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("商户不存在: %s", order.SpotID)
			return webapi.Error(common.ErrCheckInPointNotExist).Render(ec)
		}
		sdlog.Errorf("查询商户失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	order.Status = common.ProductOrderStatusCancelled
	err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusCancelled).
		Update("cancel_reason", req.Reason).
		Update("cancel_by_buyer", false).
		Update("cancel_at", time.Now().Unix()).
		Update("updated_at", time.Now()).Error
	if err != nil {
		sdlog.Errorf("取消订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 若存在支付订单，则取消支付订单
	var payOrder model.PaymentOrder
	err = ec.Nu.DB.Model(&model.PaymentOrder{}).
		Where("order_id = ?", orderID).
		First(&payOrder).Error
	if err != nil {
		sdlog.Errorf("查询支付订单失败: %v", err)
	}
	if payOrder.Status == common.OrderStatusUnpaid {
		err = ec.Nu.DB.Model(&payOrder).Update("status", common.OrderStatusCancel).Error
		if err != nil {
			sdlog.Errorf("取消支付订单失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	sdlog.Infof("订单 %s 取消成功", orderID)
	return webapi.OK(true).Render(ec)
}

func getProductPayInfo(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取支付信息")

	// 获取订单ID
	orderID := ec.QueryParams().Get("order_id")
	if orderID == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.ProductOrder
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrProductOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证订单状态
	if order.Status != common.ProductOrderStatusPendingPayment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 验证订单所属
	if order.UserID != ec.AuthData.User.ID {
		sdlog.Error("无权访问此订单")
		return webapi.Error(common.ErrUnPower).Render(ec)
	}
	if order.PayType == "FEC" {
		if order.UeOrderID != "" {
			sdlog.Infof("订单已存在，支付信息为，订单ID: %s", orderID)
			return webapi.OK(PayOrderResponse{
				OrderID: strconv.FormatInt(order.OrderID, 10),
				Paycode: ec.Nu.Config.UE.PayCode,
				Amount:  decimal.NewFromFloat(order.Amount).InexactFloat64(),
				PayType: 25,
				Unit:    16,
			}).Render(ec)
		}
		// 创建UE支付请求
		ueReq := &models.CreateOrderV2Req{
			UEReqbaseV2:    models.NewUEReqbaseV2(),
			Paycode:        ec.Nu.Config.UE.PayCode,
			BusinessTypeid: ec.Nu.Config.UE.BusinessTypeid,
			Subject:        strconv.FormatInt(order.OrderID, 10),
			Body:           order.ProductName,
			Orderno:        strconv.FormatInt(order.OrderID, 10),
			NoticeUrl:      fmt.Sprintf("%s/api/v1/ue/notify", ec.Nu.Config.UE.MyDomain),
			OrderItems: []models.OrderItem{{
				Paytype: 25,
				Amount:  order.Amount,
				Unit:    16,
			}},
		}
		ueConfig := models.UeConfigParam2{
			Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
			Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
			Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
		}

		resp, err := ue_api_v2.CreateOrderV2(*ueReq, ueConfig)
		if err != nil {
			sdlog.Errorf("[CreateOrderV2] 创建支付订单失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}

		if resp.Result <= 0 {
			sdlog.Errorf("[CreateOrderV2] 创建支付订单失败: %s", resp.Message)
			return webapi.Error(common.ErrService).Render(ec)
		}
		// 更新订单的UE订单ID
		err = ec.Nu.DB.Model(&order).Update("ue_order_id", 1).Error
		if err != nil {
			sdlog.Errorf("更新订单UE订单ID失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
		sdlog.Infof("获取支付信息成功，订单ID: %s", orderID)
		return webapi.OK(PayOrderResponseV2{
			OrderID:      ueReq.Orderno,
			Paycode:      ueReq.Paycode,
			Amount:       decimal.NewFromFloat(order.Amount).InexactFloat64(),
			PayType:      order.PayType,
			Currency:     order.Unit,
			ProductNum:   order.ProductNum,
			ProductPrice: order.ProductPrice,
			Value:        strconv.FormatFloat(order.Amount, 'f', -1, 64),
		}).Render(ec)
	} else {
		// 查询address
		var address model.AddressCheckpoint
		err = ec.Nu.DB.Model(&model.AddressCheckpoint{}).
			Where("currency = ?", order.Unit).
			Where("chain = ?", order.PayType).
			Where("is_active = ?", true).
			First(&address).Error
		if err != nil {
			sdlog.Errorf("支付方式不存在: %v", err)
			return webapi.Error(common.ErrPayMethodNotExist).Render(ec)
		}

		//验证是否已存在支付订单
		var payOrder model.PaymentOrder
		err := ec.Nu.DB.Model(&model.PaymentOrder{}).
			Where("order_id = ?", orderID).
			First(&payOrder).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 新建支付订单
				newPayOrder := model.PaymentOrder{
					OrderID:     orderID,
					Chain:       order.PayType,
					Currency:    order.Unit,
					Amount:      decimal.NewFromFloat(order.Amount).InexactFloat64(),
					Status:      common.OrderStatusUnpaid,
					FromAddress: "",
					ToAddress:   address.Address,
					CreatedAt:   time.Now(),
					PayOrderID:  "pay_" + orderID,
					Description: "",
					Metadata:    "{}",
					IsCallback:  true,
					PaidAt:      time.Now(),
					ExpiredAt:   time.Now().Add(time.Minute * 15),
					BizType:     common.PaymentBizTypeProduct,
				}
				if err = ec.Nu.DB.Create(&newPayOrder).Error; err != nil {
					sdlog.Error("创建支付订单失败", err)
					return webapi.Error(common.ErrService).Render(ec)
				}
			} else {
				sdlog.Errorf("查询支付订单失败: %v", err)
				return webapi.Error(common.ErrService).Render(ec)
			}
		}
		err = ec.Nu.DB.Model(&model.PaymentOrder{}).
			Where("order_id = ?", orderID).
			First(&payOrder).Error

		sdlog.Infof("获取支付信息成功，订单ID: %s", orderID)
		return webapi.OK(PayOrderResponseV2{
			OrderID:          payOrder.OrderID,
			Paycode:          "",
			Amount:           decimal.NewFromFloat(payOrder.Amount).InexactFloat64(),
			ProductNum:       order.ProductNum,
			ProductPrice:     decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
			PayType:          payOrder.Chain,
			Currency:         payOrder.Currency,
			Value:            ConvertAmountToValue(payOrder.Amount, int(address.TokenDecimals)),
			ReceivingAddress: payOrder.ToAddress,
			ContractAddress:  address.ContractAddress,
		}).Render(ec)
	}
}

type ConfirmProductOrderShipmentRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

// 确认发货
func confirmProductOrderShipment(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始确认发货")

	req := ConfirmProductOrderShipmentRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// spot owner check in point id
	spotOwnerID := order.SpotID
	spotOwner := model.CheckInPoint{}
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id = ?", spotOwnerID).
		First(&spotOwner).Error
	if err != nil {
		sdlog.Errorf("查询商户失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if spotOwner.OwnerID != ec.AuthData.User.ID {
		sdlog.Errorf("商户不属于当前用户: %d", spotOwnerID)
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingShipment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusPendingReceipt).Error
	if err != nil {
		sdlog.Errorf("更新订单状态失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 确认发货成功", orderID)
	return webapi.OK(true).Render(ec)
}

type ConfirmProductOrderReceiptRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

// 确认收货
func confirmProductOrderReceipt(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始确认收货")

	req := ConfirmProductOrderReceiptRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingReceipt {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}
	// 退款中不能确认收货
	if order.RefundStatus == common.ProductRefundStatusRefunding {
		sdlog.Errorf("订单退款状态不正确: %d", order.RefundStatus)
		return webapi.Error(common.ErrOrderRefundingCannotConfirmReceipt).Render(ec)
	}

	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	order.Status = common.ProductOrderStatusCompleted
	err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusCompleted).Error
	if err != nil {
		sdlog.Errorf("更新订单状态失败: %v", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	amount := decimal.NewFromFloat(order.Amount)

	// 商户收到货款 更新资产表 user_asset user_asset_detail
	userAsset := model.UserAsset{}
	err = ec.Nu.DB.Model(&model.UserAsset{}).
		Where("user_id = ?", order.SellerID).
		Where("currency = ?", order.Unit).
		First(&userAsset).Error
	if err != nil {
		sdlog.Errorf("查询商户资产失败: %v", err)
		// 如果用户资产不存在，则创建用户资产
		if err == gorm.ErrRecordNotFound {
			userAsset = model.UserAsset{
				UserID:    order.SellerID,
				Currency:  order.Unit,
				Balance:   amount.InexactFloat64(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = ec.Nu.DB.Model(&model.UserAsset{}).Create(&userAsset).Error
			if err != nil {
				sdlog.Errorf("创建商户资产失败: %v", err)
				tx.Rollback()
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			tx.Rollback()
			return webapi.Error(common.ErrService).Render(ec)
		}
	} else {
		userAsset.Balance += order.Amount
		err = ec.Nu.DB.Model(&model.UserAsset{}).
			Where("user_id = ?", order.SellerID).
			Where("currency = ?", order.Unit).
			Update("balance", userAsset.Balance).Error
		if err != nil {
			sdlog.Errorf("更新商户资产失败: %v", err)
			tx.Rollback()
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	userAssetDetail := model.UserAssetDetail{
		UserID:    order.SellerID,
		Category:  common.UserAssetCategorySold,
		Currency:  order.Unit,
		Rewards:   amount.InexactFloat64(),
		RelatedID: order.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		InviteeID: 0,
	}
	err = ec.Nu.DB.Model(&model.UserAssetDetail{}).Create(&userAssetDetail).Error
	if err != nil {
		sdlog.Errorf("插入商户资产明细失败: %v", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}
	tx.Commit()

	sdlog.Infof("订单 %s 确认收货成功", orderID)
	return webapi.OK(true).Render(ec)
}

type ApplyRefundRequest struct {
	OrderID int64  `json:"order_id"` // 订单ID
	Reason  string `json:"reason"`   // 退款原因
}

func applyRefund(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始申请退款")

	req := ApplyRefundRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 || req.Reason == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 只有待收货和待发货状态可以申请退款
	if order.Status != common.ProductOrderStatusPendingReceipt && order.Status != common.ProductOrderStatusPendingShipment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusCannotApplyRefund).Render(ec)
	}

	if order.RefundStatus != common.ProductRefundStatusNoRefund {
		sdlog.Errorf("订单退款状态不正确: %d", order.RefundStatus)
		return webapi.Error(common.ErrOrderAlreadyAppliedRefund).Render(ec)
	}

	// 开启事务
	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	if tx.Error != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	amount := decimal.NewFromFloat(order.Amount)
	// 创建退款订单
	refundOrder := model.ProductRefund{
		OrderID:   orderID,
		UserID:    ec.AuthData.User.ID,
		Amount:    amount.InexactFloat64(),
		Unit:      order.Unit,
		Status:    common.ProductRefundStatusRefunding,
		Remark:    req.Reason,
		Timestamp: time.Now().Unix(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err = tx.Create(&refundOrder).Error; err != nil {
		sdlog.Error("创建退款订单失败", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新订单退款状态为退款中
	err = tx.Model(&order).Update("refund_status", common.ProductRefundStatusRefunding).Error
	if err != nil {
		sdlog.Errorf("更新订单状态失败: %v", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("退款订单创建成功: %d", refundOrder.ID)
	return webapi.OK(true).Render(ec)
}

type HandleRefundRequest struct {
	RefundID     int64  `json:"refund_id"`     // 退款ID
	Approve      bool   `json:"approve"`       // 是否同意退款
	RejectReason string `json:"reject_reason"` // 拒绝退款原因
}

func handleRefund(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始处理退款")

	req := HandleRefundRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	refundID := req.RefundID
	if refundID == 0 {
		sdlog.Error("退款ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if !req.Approve && req.RejectReason == "" {
		sdlog.Error("拒绝退款原因不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	refund := model.ProductRefund{}
	err := ec.Nu.DB.Model(&model.ProductRefund{}).
		Where("id = ?", refundID).
		First(&refund).Error
	if err != nil {
		sdlog.Errorf("查询退款失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询订单seller_id是否等于当前用户
	order := model.ProductOrder{}
	err = ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", refund.OrderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.SellerID != ec.AuthData.User.ID {
		sdlog.Errorf("订单不属于当前用户: %d", order.SellerID)
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	if refund.Status != common.ProductRefundStatusRefunding {
		sdlog.Errorf("退款状态不正确: %d", refund.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	if order.RefundStatus != common.ProductRefundStatusRefunding {
		sdlog.Errorf("订单退款状态不正确: %d", order.RefundStatus)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	if req.Approve {
		tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
		if tx.Error != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
		// 更新订单状态为关闭
		order.Status = common.ProductOrderStatusClosed
		err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusClosed).Update("refund_status", common.ProductRefundStatusRefunded).Error
		if err != nil {
			sdlog.Errorf("更新订单状态失败: %v", err)
			tx.Rollback()
			return webapi.Error(common.ErrService).Render(ec)
		}

		// 更新订单退款状态为退款成功
		refund.Status = common.ProductRefundStatusRefunded
		err = ec.Nu.DB.Model(&refund).Update("status", common.ProductRefundStatusRefunded).Error
		if err != nil {
			sdlog.Errorf("更新订单退款状态失败: %v", err)
			tx.Rollback()
			return webapi.Error(common.ErrService).Render(ec)
		}

		// 更新用户资产,币种全部转为大写，如果原记录不存在，则插入一条新记录
		unit := strings.ToUpper(order.Unit)
		userAsset := model.UserAsset{}

		amount := decimal.NewFromFloat(refund.Amount)
		err = ec.Nu.DB.Model(&model.UserAsset{}).
			Where("user_id = ?", order.UserID).
			Where("currency = ?", unit).
			First(&userAsset).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				userAsset = model.UserAsset{
					UserID:   order.UserID,
					Currency: unit,
					Balance:  amount.InexactFloat64(),
				}

				err = ec.Nu.DB.Model(&model.UserAsset{}).Create(&userAsset).Error
				if err != nil {
					sdlog.Errorf("创建用户资产失败: %v", err)
					tx.Rollback()
					return webapi.Error(common.ErrService).Render(ec)
				}
			} else {
				sdlog.Errorf("查询用户资产失败: %v", err)
				tx.Rollback()
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			userBalance := decimal.NewFromFloat(userAsset.Balance).Add(decimal.NewFromFloat(amount.InexactFloat64()))
			userAsset.Balance = userBalance.InexactFloat64()
			err = ec.Nu.DB.Model(&model.UserAsset{}).
				Where("user_id = ?", order.UserID).
				Where("currency = ?", unit).
				Update("balance", userBalance.InexactFloat64()).Error
			if err != nil {
				sdlog.Errorf("更新用户资产失败: %v", err)
				tx.Rollback()
				return webapi.Error(common.ErrService).Render(ec)
			}
		}

		// 插入资产明细
		assetDetail := model.UserAssetDetail{
			UserID:    order.UserID,
			RelatedID: refund.ID,
			Category:  common.UserAssetCategoryRefund,
			Rewards:   amount.InexactFloat64(),
			Currency:  unit,
			InviteeID: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = ec.Nu.DB.Model(&model.UserAssetDetail{}).Create(&assetDetail).Error
		if err != nil {
			sdlog.Errorf("插入资产明细失败: %v", err)
			tx.Rollback()
			return webapi.Error(common.ErrService).Render(ec)
		}

		// 商户资产明细 扣除对应余额 并插入一条记录
		// sellerUserAsset := model.UserAsset{}
		// err = ec.Nu.DB.Model(&model.UserAsset{}).
		// 	Where("user_id = ?", order.SellerID).
		// 	Where("currency = ?", order.Unit).
		// 	First(&sellerUserAsset).Error
		// if err != nil {
		// 	sdlog.Errorf("查询商户资产失败: %v", err)
		// 	tx.Rollback()
		// 	return webapi.Error(common.ErrService).Render(ec)
		// }

		// sellerUserAsset.Balance = sellerUserAsset.Balance - refund.Amount
		// err = ec.Nu.DB.Model(&model.UserAsset{}).
		// 	Where("user_id = ?", order.SellerID).
		// 	Where("currency = ?", order.Unit).
		// 	Update("balance", sellerUserAsset.Balance).Error
		// if err != nil {
		// 	sdlog.Errorf("更新商户资产失败: %v", err)
		// 	tx.Rollback()
		// 	return webapi.Error(common.ErrService).Render(ec)
		// }

		// sellerAssetDetail := model.UserAssetDetail{
		// 	UserID:    order.SellerID,
		// 	Category:  common.UserAssetCategoryRefundToUser,
		// 	Currency:  order.Unit,
		// 	Rewards:   refund.Amount,
		// 	RelatedID: refund.ID,
		// 	CreatedAt: time.Now(),
		// 	UpdatedAt: time.Now(),
		// 	InviteeID: 0,
		// }
		// err = ec.Nu.DB.Model(&model.UserAssetDetail{}).Create(&sellerAssetDetail).Error
		// if err != nil {
		// 	sdlog.Errorf("插入商户资产明细失败: %v", err)
		// 	tx.Rollback()
		// 	return webapi.Error(common.ErrService).Render(ec)
		// }
		tx.Commit()
	} else {
		// 更新退款状态为退款拒绝
		refund.Status = common.ProductRefundStatusRefundRejected
		refund.Remark = req.RejectReason
		err = ec.Nu.DB.Model(&refund).Update("status", common.ProductRefundStatusRefundRejected).Update("remark", req.RejectReason).Error
		if err != nil {
			sdlog.Errorf("更新退款状态失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
		// 更新订单退款状态未无退款
		err = ec.Nu.DB.Model(&order).Update("refund_status", common.ProductRefundStatusRefundRejected).Error
		if err != nil {
			sdlog.Errorf("更新订单退款状态失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	sdlog.Infof("退款 %s 处理成功", refundID)
	return webapi.OK(true).Render(ec)
}

type UpdateProductOrderPriceRequest struct {
	OrderID int64   `json:"order_id"` // 订单ID
	Price   float64 `json:"price"`    // 价格
}

func updateProductOrderPrice(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始修改订单价格")

	req := UpdateProductOrderPriceRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 || req.Price <= 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.SellerID != ec.AuthData.User.ID {
		sdlog.Errorf("订单不属于当前用户: %d", order.SellerID)
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingPayment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	if order.PayType == "FEC" {
		sdlog.Errorf("fec支付不支持修改价格: %s", order.PayType)
		return webapi.Error(common.ErrFECCannotUpdatePrice).Render(ec)
	}
	productAmount := req.Price * float64(order.ProductNum)
	err = ec.Nu.DB.Model(&order).Update("product_price", req.Price).Update("amount", productAmount).Error
	if err != nil {
		sdlog.Errorf("更新订单价格失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	productAmount = decimal.NewFromFloat(req.Price).InexactFloat64() * decimal.NewFromFloat(float64(order.ProductNum)).InexactFloat64()
	err = ec.Nu.DB.Model(&order).Update("product_price", req.Price).Update("amount", productAmount).Error
	if err != nil {
		sdlog.Errorf("更新订单价格失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 同时修改payment order 金额
	paymentOrder := model.PaymentOrder{}
	err = ec.Nu.DB.Model(&model.PaymentOrder{}).
		Where("order_id = ?", orderID).
		First(&paymentOrder).Error
	if err != nil {
		sdlog.Errorf("查询支付订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if paymentOrder.Status != common.OrderStatusUnpaid {
		sdlog.Errorf("支付订单状态不正确: %s", paymentOrder.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}
	paymentOrder.Amount = decimal.NewFromFloat(req.Price).InexactFloat64()
	err = ec.Nu.DB.Model(&paymentOrder).Update("amount", paymentOrder.Amount).Error
	if err != nil {
		sdlog.Errorf("更新支付订单金额失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 价格修改成功", orderID)
	return webapi.OK(true).Render(ec)
}

type UpdateProductOrderAddressRequest struct {
	OrderID int64  `json:"order_id"` // 订单ID
	Phone   string `json:"phone"`    // 电话
	Name    string `json:"name"`     // 姓名
	Address string `json:"address"`  // 地址
}

func updateProductOrderAddress(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始修改订单地址")

	req := UpdateProductOrderAddressRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 || req.Phone == "" || req.Name == "" || req.Address == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.UserID != ec.AuthData.User.ID {
		sdlog.Errorf("订单不属于当前用户: %d", order.UserID)
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingPayment && order.Status != common.ProductOrderStatusPendingShipment && order.Status != common.ProductOrderStatusPaymentProcessing {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	order.ContactPhone = req.Phone
	order.ContactName = req.Name
	order.ContactAddress = req.Address
	err = ec.Nu.DB.Model(&order).Update("contact_phone", req.Phone).Update("contact_name", req.Name).Update("contact_address", req.Address).Error
	if err != nil {
		sdlog.Errorf("更新订单地址失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 地址修改成功", orderID)
	return webapi.OK(true).Render(ec)
}

type DeleteProductOrderRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

func deleteProductOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始删除订单")

	req := DeleteProductOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		First(&order).Error
	if err != nil {
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.UserID != ec.AuthData.User.ID {
		sdlog.Errorf("订单不属于当前用户: %d", order.UserID)
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	if order.Status != common.ProductOrderStatusCancelled && order.Status != common.ProductOrderStatusCompleted && order.Status != common.ProductOrderStatusClosed {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}
	// 标记为已经删除，更新删除时间
	order.IsDeleted = true
	order.DeletedAt = time.Now().Unix()

	err = ec.Nu.DB.Model(&order).Update("is_deleted", true).Update("deleted_at", time.Now().Unix()).Error
	if err != nil {
		sdlog.Errorf("删除订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 删除成功", orderID)
	return webapi.OK(true).Render(ec)
}

type PayReportRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

func payReport(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始支付上报")

	req := PayReportRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	order := model.ProductOrder{}
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %d", orderID)
			return webapi.Error(common.ErrOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if order.Status != common.ProductOrderStatusPendingPayment {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	order.Status = common.ProductOrderStatusPaymentProcessing
	err = ec.Nu.DB.Model(&order).Update("status", common.ProductOrderStatusPaymentProcessing).Error
	if err != nil {
		sdlog.Errorf("更新订单状态失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 支付上报成功", orderID)
	return webapi.OK(true).Render(ec)
}
