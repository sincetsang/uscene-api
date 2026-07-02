package service

import (
	"TMA/pkg/cache/memory"
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CheckInInfo struct {
	TotalCount int64                `json:"total_count"` // 地标总数
	Points     []CheckInPointStruct `json:"points"`
}

type CheckInPointStruct struct {
	Distance               float64                          `json:"distance"`       // 与用户的距离
	CategoryEn             string                           `json:"category_en"`    // 地标分类英文
	Category               string                           `json:"category"`       // 地标分类
	CategoryImage          string                           `json:"category_image"` // 地标分类图片
	Claim                  float64                          `json:"claim"`
	AdvertisementIsClaimed bool                             `json:"advertisement_is_claimed" gorm:"-"` // 广告是否已领取
	Point                  *db.CheckInPointWithOwnerV2      `json:"point"`                             // 地标详情
	PointImages            []model.CheckInPointImage        `gorm:"-"  json:"point_images"`
	Advertisement          *model.CheckInPointAdvertisement `json:"advertisement" gorm:"check_in_point_advertisement"` // 广告详情
	Product                []ProductInfoWithImage           `gorm:"-"  json:"product"`
	MinPrice               string                           `json:"min_price"`
	MinPriceCurrency       string                           `json:"min_price_currency"`
	Products               []ProductInfoWithImage           `gorm:"-"  json:"products"`
}

type ByDistance []CheckInPointStruct

func (a ByDistance) Len() int           { return len(a) }
func (a ByDistance) Less(i, j int) bool { return a[i].Distance < a[j].Distance }
func (a ByDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ByReward []CheckInPointStruct

func (a ByReward) Len() int      { return len(a) }
func (a ByReward) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByReward) Less(i, j int) bool {
	// 处理空指针情况
	if a[i].Advertisement == nil && a[j].Advertisement == nil {
		return false
	}
	if a[i].Advertisement == nil {
		return false
	}
	if a[j].Advertisement == nil {
		return true
	}
	return a[i].Advertisement.RewardAmount > a[j].Advertisement.RewardAmount
}

type ByTime []CheckInPointStruct

func (a ByTime) Len() int { return len(a) }
func (a ByTime) Less(i, j int) bool {
	return a[i].Point.CreatedAt.After(a[j].Point.CreatedAt)
}
func (a ByTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByLike []CheckInPointStruct

func (a ByLike) Len() int           { return len(a) }
func (a ByLike) Less(i, j int) bool { return a[i].Point.LikeCount > a[j].Point.LikeCount }
func (a ByLike) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func SpotListV2(ctx context.Context, nu *nucl.Nucleus, uid int64, radius int64, latitudeStr string, longitudeStr string, isOwner int64, keyword string, category string, sortBy string, hasAdvertisement int64) (*CheckInInfo, error) {
	// radius = 100000
	if latitudeStr == "" || longitudeStr == "" {
		return nil, common.ErrLocationLatLonErr
	}
	latitude := sdparse.Float64Def(latitudeStr, 0.0)
	longitude := sdparse.Float64Def(longitudeStr, 0.0)

	result := CheckInInfo{}
	visableDistanceStr := db.GetConfigByKey(nu.DB.WithContext(ctx), common.ConfigCheckPointVisibleDistance, "10000")
	visableDistance_, _ := sdparse.Float64(*visableDistanceStr)

	var checkPoints []*db.CheckInPointWithOwnerV2
	distance := float64(radius)
	if radius == 0 {
		distance = 80000000
	} else if radius == -1 {
		distance = visableDistance_
	}

	latMin, latMax, lonMin, lonMax := getBoundingBox(latitude, longitude, distance/1000)
	sdlog.Infof("latMin: %f, latMax: %f, lonMin: %f, lonMax: %f", latMin, latMax, lonMin, lonMax)
	tmpOwnerId := int64(0)
	if isOwner == 1 {
		tmpOwnerId = uid
	}
	checkPoints_, errC := db.GetAllValidCheckPointsByLaLoV2(nu.DB.WithContext(ctx), latMin, latMax, lonMin, lonMax, tmpOwnerId, keyword, uid, category, hasAdvertisement)
	if errC != nil {
		return &result, errC
	}
	checkPoints = checkPoints_

	var pointIdArr = make([]int64, 0)
	spotIds := make([]int64, len(checkPoints))
	pointsArr := []CheckInPointStruct{}

	for i, point := range checkPoints {
		spotIds[i] = point.ID
		pointIdArr = append(pointIdArr, point.ID)
		pointDistance := float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude))
		if pointDistance < distance {
			redis.AddClaim(ctx, nu, point.OwnerID, fmt.Sprint(point.ID), 1)
			claim, _ := redis.GetClaim(ctx, nu, point.OwnerID, fmt.Sprint(point.ID))
			checkInfo := CheckInPointStruct{
				Distance: float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude)),
				Point:    point,
				Category: point.Category,
				Claim:    claim,
			}
			pointsArr = append(pointsArr, checkInfo)
		}
	}
	if sortBy == "distance" {
		sort.Sort(ByDistance(pointsArr))
	} else if sortBy == "reward" {
		sort.Sort(ByReward(pointsArr))
	} else if sortBy == "time" {
		sort.Sort(ByTime(pointsArr))
	} else if sortBy == "like" {
		sort.Sort(ByLike(pointsArr))
	} else {
		sort.Sort(ByDistance(pointsArr))
	}
	// 查询广告领取记录
	var claimRecords []model.UserAdvertisementClaimRecord
	err := nu.DB.WithContext(ctx).Model(&model.UserAdvertisementClaimRecord{}).
		Where("user_id = ?", uid).
		Find(&claimRecords).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 构建广告ID到领取状态的映射
	claimedMap := make(map[int64]bool)
	for _, record := range claimRecords {
		claimedMap[record.AdvertisementID] = true
	}

	checkCountArr, errCC := db.SpotUserCheckedByIds(nu.DB.WithContext(ctx), uid, pointIdArr)
	if errCC != nil {
		return nil, common.ErrUnknown
	}
	var checkCountDic = make(map[int64]db.UserSpotChecked, 0)
	for _, item := range checkCountArr {
		checkCountDic[item.PointId] = item
	}

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err = nu.DB.WithContext(ctx).Model(&model.SpotCategory{}).Find(&categories).Error
	if err != nil {
		return nil, common.ErrService
	}
	categoryMap := make(map[string]*model.SpotCategory)
	for _, cat := range categories {
		categoryMap[cat.Type] = cat
	}

	// 批量查询广告信息
	var advertisements []model.CheckInPointAdvertisement
	err = nu.DB.WithContext(ctx).Model(&model.CheckInPointAdvertisement{}).
		Where("check_in_point_id IN ? AND status = 1", spotIds).
		Where("start_time <= ? AND end_time >= ?", time.Now(), time.Now()).
		Find(&advertisements).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 构建地标ID到广告的映射
	advertisementMap := make(map[int64]model.CheckInPointAdvertisement)
	for i, ad := range advertisements {
		advertisementMap[ad.CheckInPointID] = advertisements[i]
	}

	// 批量查询产品信息
	products, err := getFirstProductBySpotIds(ctx, nu, spotIds, []int64{common.SpotAuditStatusApproved, common.SpotAuditStatusPending})
	if err != nil {
		return nil, err
	}

	productIds := make([]int64, 0)
	for _, p := range products {
		productIds = append(productIds, int64(p.ID))
	}

	productImageMap, err := GetProductImageMapByProductIds(ctx, nu, productIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		return nil, err
	}

	// // // 构建地标ID到产品列表的映射
	// productWithImageMap := make(map[int64][]ProductInfoWithImage)
	// for _, product := range products {
	// 	if productImages, exists := productImageMap[int64(product.ID)]; exists {
	// 		productWithImageMap[int64(product.ID)] = append(productWithImageMap[int64(product.ID)], ProductInfoWithImage{
	// 			Product: product,
	// 			HeadImg: productImages[0].ImageURL,
	// 			Images:  productImages,
	// 		})
	// 	} else {
	// 		productWithImageMap[int64(product.ID)] = []ProductInfoWithImage{ProductInfoWithImage{Product: product}}
	// 	}
	// }

	productMap := make(map[int64]ProductInfoWithImage)
	for _, p := range products {
		if productImages, exists := productImageMap[int64(p.ID)]; exists {
			productMap[int64(p.CheckInPointID)] = ProductInfoWithImage{
				Product: p,
				HeadImg: productImages[0].ImageURL,
				Images:  productImages,
			}
		} else {
			productMap[int64(p.CheckInPointID)] = ProductInfoWithImage{
				Product: p,
				HeadImg: "",
				Images:  make([]model.ProductImage, 0),
			}
		}
	}

	// 批量查询地标图片
	var pointImages []model.CheckInPointImage
	err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ? and status = 1", spotIds).
		Order("check_in_point_id, sort_index").
		Find(&pointImages).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// // 构建地标ID到图片列表的映射
	pointImagesMap := make(map[int64][]model.CheckInPointImage)
	for _, img := range pointImages {
		pointImagesMap[img.CheckInPointID] = append(pointImagesMap[img.CheckInPointID], img)
	}

	productPriceMap := make(map[int64]ProductPrice)
	for _, p := range products {
		if p.AmountUsd > 0 {
			productPriceMap[p.CheckInPointID] = ProductPrice{
				Price:    p.AmountUsd,
				Currency: "USDT",
			}
		} else if p.Amount > 0 {
			productPriceMap[p.CheckInPointID] = ProductPrice{
				Price:    p.Amount,
				Currency: "FEC",
			}
		}
	}

	finalArr := []CheckInPointStruct{}
	for _, point := range pointsArr {
		if ad, exists := advertisementMap[point.Point.ID]; exists {
			point.Advertisement = &ad
			point.AdvertisementIsClaimed = claimedMap[ad.ID]
		} else {
			point.AdvertisementIsClaimed = false
		}
		category, exists := memory.CategoryMemoryCache.GetCategoryList()[point.Category]
		if exists {
			point.CategoryImage = category.ImageURL
			if common.GetRequestLanguage() == common.LanguageZh {
				point.Category = category.LabelZh
			} else {
				point.Category = category.Label
			}
			point.CategoryEn = category.Type
		}
		// 设置地标图片
		if images, exists := pointImagesMap[point.Point.ID]; exists {
			point.PointImages = images
		} else {
			point.PointImages = []model.CheckInPointImage{}
		}
		if point.Point.LandmarkImage == "" && len(point.PointImages) > 0 {
			point.Point.LandmarkImage = point.PointImages[0].ImageURL
		}
		// 设置分类图片
		if cat, exists := categoryMap[point.Point.Category]; exists {
			// lan := common.GetRequestLanguage()
			// if lan == common.LanguageZh {
			// 	point.Point.Category = cat.LabelZh
			// } else if lan == common.LanguageZhTw {
			// 	point.Point.Category = cat.LabelZhTw
			// } else if lan == common.LanguageRu {
			// 	point.Point.Category = cat.LabelRu
			// }
			point.CategoryImage = cat.ImageURL
		}
		// TODO 临时处理 后续去掉
		point.AdvertisementIsClaimed = true
		if _, exists := productPriceMap[point.Point.ID]; exists {
			if productPriceMap[point.Point.ID].Price == 0 {
				point.MinPrice = ""
				point.MinPriceCurrency = ""
			} else {
				point.MinPrice = fmt.Sprintf("%f", productPriceMap[point.Point.ID].Price)
				point.MinPriceCurrency = productPriceMap[point.Point.ID].Currency
			}
		} else {
			point.MinPrice = ""
			point.MinPriceCurrency = ""
		}

		if products, exists := productMap[point.Point.ID]; exists {
			point.Products = []ProductInfoWithImage{products}
		}
		finalArr = append(finalArr, point)
	}
	result.Points = finalArr
	result.TotalCount = int64(len(finalArr))
	return &result, nil
}

type GeocodeResponse struct {
	Results []struct {
		FormattedAddress  string `json:"formatted_address"`
		AddressComponents []struct {
			LongName string   `json:"long_name"`
			Types    []string `json:"types"`
		} `json:"address_components"`
	} `json:"results"`
	Status string `json:"status"`
}

// 检查一个切片是否包含指定的值
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// GetAddressInfo 根据经纬度获取详细地址及省市信息
func GetAddressInfo(googleMapsApiKey string, lat, lng float64) (string, string, string, string, error) {
	if len(strings.TrimSpace(googleMapsApiKey)) < 20 {
		sdlog.Warnf("[GetAddressInfo] skip reverse geocoding because google_maps_key is not configured correctly")
		return "", "", "", "", nil
	}

	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s&language=en", lat, lng, googleMapsApiKey)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", "", fmt.Errorf("请求错误: %v", err)
	}
	defer resp.Body.Close()

	var data GeocodeResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", "", "", fmt.Errorf("解析错误: %v", err)
	}

	if data.Status == "OK" && len(data.Results) > 0 {
		result := data.Results[0]

		// 提取国家、地区、城市信息
		var country, province, city string
		for _, component := range result.AddressComponents {
			if contains(component.Types, "country") {
				country = component.LongName
			}
			if contains(component.Types, "administrative_area_level_1") {
				province = component.LongName
			}
			if contains(component.Types, "locality") {
				city = component.LongName
			}
		}

		// 如果城市为空，使用省份作为城市
		if city == "" {
			city = province
		}

		// 处理城市名称中的 "Shi" 替换为 "City"
		if strings.Contains(city, "Shi") {
			city = strings.Replace(city, "Shi", "City", -1)
		}

		return country, province, city, result.FormattedAddress, nil
	}

	if data.Status == "REQUEST_DENIED" {
		sdlog.Warnf("[GetAddressInfo] reverse geocoding denied by Google Maps API")
		return "", "", "", "", nil
	}

	return "", "", "", "", fmt.Errorf("错误: %s", data.Status)
}

func GetSpotImagesBySpotIds(ctx context.Context, nu *nucl.Nucleus, spotIds []int64, status []int) ([]model.CheckInPointImage, error) {
	query := nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ?", spotIds)
	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}
	query = query.Order("check_in_point_id").Order("sort_index")
	var spotImages []model.CheckInPointImage
	err := query.Find(&spotImages).Error
	if err != nil {
		return nil, err
	}
	return spotImages, nil
}

func GetSpotImageMapBySpotIds(ctx context.Context, nu *nucl.Nucleus, spotIds []int64, status []int) (map[int64][]model.CheckInPointImage, error) {
	spotImages, err := GetSpotImagesBySpotIds(ctx, nu, spotIds, status)
	if err != nil {
		return nil, err
	}
	spotImageMap := make(map[int64][]model.CheckInPointImage)
	for _, spotImage := range spotImages {
		spotImageMap[spotImage.CheckInPointID] = append(spotImageMap[spotImage.CheckInPointID], spotImage)
	}
	return spotImageMap, nil
}
