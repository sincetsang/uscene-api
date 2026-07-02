package service

import (
	"TMA/pkg/cache/memory"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"
)

type CheckInInfoV4 struct {
	TotalCount int64                  `json:"total_count"` // 地标总数
	Points     []CheckInPointStructV4 `json:"points"`      // 地标列表
}

type CheckInPointStructV4 struct {
	Distance      float64                          `json:"distance"`                                          // 与用户的距离
	CategoryImage string                           `json:"category_image"`                                    // 地标分类图片
	Point         *db.CheckInPointWithOwnerV4      `json:"point"`                                             // 地标详情
	Advertisement *model.CheckInPointAdvertisement `json:"advertisement" gorm:"check_in_point_advertisement"` // 广告详情
}

type ByDistanceV4 []CheckInPointStructV4

func (a ByDistanceV4) Len() int           { return len(a) }
func (a ByDistanceV4) Less(i, j int) bool { return a[i].Distance < a[j].Distance }
func (a ByDistanceV4) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ByRewardV4 []CheckInPointStructV4

func (a ByRewardV4) Len() int      { return len(a) }
func (a ByRewardV4) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRewardV4) Less(i, j int) bool {
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

type ByTimeV4 []CheckInPointStructV4

func (a ByTimeV4) Len() int { return len(a) }
func (a ByTimeV4) Less(i, j int) bool {
	return a[i].Point.CreatedAt.After(a[j].Point.CreatedAt)
}
func (a ByTimeV4) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func SpotListV4(ctx context.Context, nu *nucl.Nucleus, uid int64, radius int64, latitudeStr string, longitudeStr string, isOwner int64, keyword string, category string, sortBy string, hasAdvertisement, page, page_size int64) (*CheckInInfoV4, error) {

	if latitudeStr == "" || longitudeStr == "" {
		return nil, common.ErrLocationLatLonErr
	}
	latitude := sdparse.Float64Def(latitudeStr, 0.0)
	longitude := sdparse.Float64Def(longitudeStr, 0.0)

	result := CheckInInfoV4{}

	var checkPoints []*db.CheckInPointWithOwnerV4
	distance := float64(radius)
	if radius == 0 {
		distance = 1000000
	} else if radius == -1 {
		distance = 500000
	}

	latMin, latMax, lonMin, lonMax := getBoundingBox(latitude, longitude, distance/1000)
	sdlog.Infof("latMin: %f, latMax: %f, lonMin: %f, lonMax: %f", latMin, latMax, lonMin, lonMax)
	tmpOwnerId := int64(0)
	if isOwner == 1 {
		tmpOwnerId = uid
	}
	checkPoints_, count, errC := db.GetAllValidCheckPointsByLaLoV4(nu.DB.WithContext(ctx), latMin, latMax, lonMin, lonMax, tmpOwnerId, keyword, uid, category, hasAdvertisement, page, page_size)
	if errC != nil {
		return &result, errC
	}
	checkPoints = checkPoints_

	var pointIdArr = make([]int64, 0)
	spotIds := make([]int64, len(checkPoints))
	pointsArr := []CheckInPointStructV4{}

	for i, point := range checkPoints {
		spotIds[i] = point.ID
		pointIdArr = append(pointIdArr, point.ID)
		pointDistance := float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude))
		if pointDistance < distance {
			checkInfo := CheckInPointStructV4{
				Distance: pointDistance,
				Point:    point,
			}
			pointsArr = append(pointsArr, checkInfo)
		}
	}

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err := nu.DB.WithContext(ctx).Model(&model.SpotCategory{}).Find(&categories).Error
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

	// 批量查询地标图片
	var pointImages []model.CheckInPointImage
	err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ? and status = 1", spotIds).
		Order("check_in_point_id, sort_index").
		Find(&pointImages).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 构建地标ID到图片列表的映射
	pointImagesMap := make(map[int64][]model.CheckInPointImage)
	for _, img := range pointImages {
		pointImagesMap[img.CheckInPointID] = append(pointImagesMap[img.CheckInPointID], img)
	}

	finalArr := []CheckInPointStructV4{}
	for _, point := range pointsArr {
		if ad, exists := advertisementMap[point.Point.ID]; exists {
			point.Advertisement = &ad
		}
		if point.Point.LandmarkImage == "" {
			if images, exists := pointImagesMap[point.Point.ID]; exists && len(images) > 0 {
				point.Point.LandmarkImage = images[0].ImageURL
			}
		}
		// 设置分类图片
		if cat, exists := categoryMap[point.Point.Category]; exists {
			point.CategoryImage = cat.ImageURL
		}
		finalArr = append(finalArr, point)
	}
	if sortBy == "distance" {
		sort.Sort(ByDistanceV4(finalArr))
	} else if sortBy == "reward" {
		sort.Sort(ByRewardV4(finalArr))
	} else if sortBy == "time" {
		sort.Sort(ByTimeV4(finalArr))
	} else {
		sort.Sort(ByDistanceV4(finalArr))
	}
	result.Points = finalArr
	result.TotalCount = count
	return &result, nil
}

type CheckInInfoPithy struct {
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	SpotId        int64   `json:"spot_id"`
	Title         string  `json:"title"`
	Category      string  `json:"category"`
	CategoryEn    string  `json:"category_en"`
	CategoryImage string  `json:"category_image"`
	OwnerId       int64   `json:"owner_id"`
	OwnerUsername string  `json:"owner_username"`
	OwnerAvatar   string  `json:"owner_avatar"`
	LandmarkImage string  `json:"landmark_image"`
	Country       string  `gorm:"country" json:"-"`
	IsChina       bool    `json:"is_china"`
	Location      string  `json:"location"`
	LikeCount     int64   `json:"like_count"`

	CheckinCount        int64 `json:"checkin_count"`
	CheckinLikeCount    int64 `json:"checkin_like_count"`
	CheckinCommentCount int64 `json:"checkin_comment_count"`
	VisitCount          int64 `json:"visit_count"`
	HotValue            int64 `json:"hot_value"`
	HotRank             int64 `json:"hot_rank"`
}

type CheckInInfoPithyWithProduct struct {
	CheckInInfoPithy
	MinPrice         string                 `json:"min_price"`
	MinPriceCurrency string                 `json:"min_price_currency"`
	Products         []ProductInfoWithImage `json:"products"`
}

type ProductInfoWithImage struct {
	model.Product
	HeadImg string               `gorm:"column:image" json:"head_img"` // 产品头图
	Images  []model.ProductImage `json:"images"`                       // 产品图片列表
}

// GridCell 表示一个网格单元
type GridCell struct {
	// GridId    string             `json:"grid_id"`          // 网格ID
	Latitude  float64                       `json:"latitude"`         // 网格中心点纬度
	Longitude float64                       `json:"longitude"`        // 网格中心点经度
	Count     string                        `json:"count"`            // 网格内的地标数量
	Type      string                        `json:"type"`             // 网格类型
	Points    []CheckInInfoPithyWithProduct `json:"points,omitempty"` // 当zoom级别较大时，包含具体的地标点信息
}

// GridResponse 网格化查询的响应
type GridResponse struct {
	Grids []GridCell `json:"grids"` // 网格列表
}

// GridRange 表示网格的经纬度范围
type GridRange struct {
	MinLat  float64 // 最小纬度
	MaxLat  float64 // 最大纬度
	MinLng  float64 // 最小经度
	MaxLng  float64 // 最大经度
	LatSize float64 // 纬度网格大小
	LngSize float64 // 经度网格大小
}

// GridInfo 表示网格的信息
type GridInfo struct {
	GridLat  int
	GridLng  int
	Count    int64
	Points   []CheckInInfoPithyWithProduct
	TotalLat float64
	TotalLng float64
}

// calculateGridKey 计算点所在的网格key
func calculateGridKey(point CheckInInfoPithyWithProduct, gridRange GridRange) string {
	// 计算点所在的网格索引
	latIndex := int(math.Floor((point.Latitude - gridRange.MinLat) / gridRange.LatSize))
	lngIndex := int(math.Floor((point.Longitude - gridRange.MinLng) / gridRange.LngSize))

	return fmt.Sprintf("%d,%d", latIndex, lngIndex)
}

// addPointToGrid 将点添加到网格中
func addPointToGrid(gridMap map[string]GridInfo, point CheckInInfoPithyWithProduct, gridKey string) map[string]GridInfo {
	if grid, exists := gridMap[gridKey]; exists {
		grid.Count++
		grid.Points = append(grid.Points, point)
		grid.TotalLat += point.Latitude
		grid.TotalLng += point.Longitude
		gridMap[gridKey] = grid
	} else {
		// 从gridKey中解析出网格坐标
		var latIndex, lngIndex int
		fmt.Sscanf(gridKey, "%d,%d", &latIndex, &lngIndex)

		gridMap[gridKey] = GridInfo{
			GridLat:  latIndex,
			GridLng:  lngIndex,
			Count:    1,
			Points:   []CheckInInfoPithyWithProduct{point},
			TotalLat: point.Latitude,
			TotalLng: point.Longitude,
		}
	}
	return gridMap
}

// calculateGridCenter 计算网格的中心点
func calculateGridCenter(grid GridInfo) (float64, float64) {
	if len(grid.Points) == 0 {
		return 0, 0
	}

	// 找到所有点的最大最小经纬度
	minLat := grid.Points[0].Latitude
	maxLat := grid.Points[0].Latitude
	minLng := grid.Points[0].Longitude
	maxLng := grid.Points[0].Longitude

	for _, point := range grid.Points {
		if point.Latitude < minLat {
			minLat = point.Latitude
		}
		if point.Latitude > maxLat {
			maxLat = point.Latitude
		}
		if point.Longitude < minLng {
			minLng = point.Longitude
		}
		if point.Longitude > maxLng {
			maxLng = point.Longitude
		}
	}

	// 计算中心点
	return (minLat + maxLat) / 2, (minLng + maxLng) / 2
}

type ProductPrice struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

// calculateOptimalGridSize 根据可视区域计算网格的经纬度范围
func calculateOptimalGridSize(swLat, swLng, neLat, neLng float64) GridRange {
	// 计算可视区域的大小（度）
	latDiff := math.Abs(neLat - swLat)
	lngDiff := math.Abs(neLng - swLng)

	// 根据可视区域大小确定网格数量
	var gridCount float64
	switch {
	case latDiff > 20 || lngDiff > 20: // 大于20度（国家级）
		gridCount = 200.0
	case latDiff > 10 || lngDiff > 10: // 大于10度（省级）
		gridCount = 200.0
	case latDiff > 5 || lngDiff > 5: // 大于5度（市级）
		gridCount = 50.0
	case latDiff > 2 || lngDiff > 2: // 大于2度（区级）
		gridCount = 20.0
	case latDiff > 1 || lngDiff > 1: // 大于1度（街道级）
		gridCount = 50.0
	default: // 小于1度
		gridCount = 20.0
	}
	// 计算每个网格的经纬度范围
	latSize := latDiff / math.Sqrt(gridCount)
	lngSize := lngDiff / math.Sqrt(gridCount)

	return GridRange{
		MinLat:  swLat,
		MaxLat:  neLat,
		MinLng:  swLng,
		MaxLng:  neLng,
		LatSize: latSize,
		LngSize: lngSize,
	}
}

type HotItem struct {
	SpotId   int64 `json:"spot_id"`
	HotValue int64 `json:"hot_value"`
}

// SpotListGrid 获取网格化的地标数据
func SpotListGrid(ctx context.Context, nu *nucl.Nucleus, uid int64, swLat, swLng, neLat, neLng float64, isOwner int64, keyword, category string) (*GridResponse, error) {
	// 首先查询可视区域内的总点数
	latMin := math.Min(swLat, neLat)
	latMax := math.Max(swLat, neLat)
	lngMin := math.Min(swLng, neLng)
	lngMax := math.Max(swLng, neLng)
	swLat = latMin
	neLat = latMax
	swLng = lngMin
	neLng = lngMax
	var totalCount int64
	countQuery := nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
		Where("latitude >= ? AND latitude <= ? AND longitude >= ? AND longitude <= ? AND status = 1 and is_active=1 and audit_status=1",
			swLat, neLat, swLng, neLng)
	// 添加条件查询
	if isOwner == 1 {
		countQuery = countQuery.Where("owner_id = ?", uid)
	}
	if keyword != "" {
		countQuery = countQuery.Where("title LIKE ?", "%"+keyword+"%")
	}
	if category != "" {
		countQuery = countQuery.Where("category = ?", category)
	}

	err := countQuery.Count(&totalCount).Error
	if err != nil {
		return nil, err
	}

	response := &GridResponse{
		Grids: make([]GridCell, 0),
	}

	// 查询所有点信息
	query := nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
		Select("check_in_point.id as spot_id, check_in_point.latitude, check_in_point.longitude, check_in_point.title, check_in_point.category, check_in_point.owner_id, user.nickname as owner_username, user.avatar_url as owner_avatar, MIN(check_in_point_image.image_url) as landmark_image, check_in_point.location, check_in_point.country, check_in_point.like_count, check_in_point.checkin_count, check_in_point.checkin_like_count, check_in_point.checkin_comment_count, check_in_point.checkin_visit_count, check_in_point.checkin_count * 4 + check_in_point.checkin_like_count * 3 + check_in_point.checkin_comment_count * 2 + check_in_point.checkin_visit_count * 1 as hot_value").
		Joins("left join user on user.user_id = check_in_point.owner_id").
		Joins("left join check_in_point_image on check_in_point_image.check_in_point_id = check_in_point.id and check_in_point_image.status = 1").
		Where("check_in_point.status = 1 AND check_in_point.is_active = 1 AND check_in_point.audit_status = 1 AND check_in_point.latitude BETWEEN ? AND ? AND check_in_point.longitude BETWEEN ? AND ?",
			swLat, neLat, swLng, neLng).
		Group("check_in_point.id, check_in_point.latitude, check_in_point.longitude, check_in_point.title, check_in_point.category, check_in_point.owner_id, user.nickname, user.avatar_url, check_in_point.location")

	// 添加条件查询
	if isOwner == 1 {
		query = query.Where("check_in_point.owner_id = ?", uid)
	}
	if keyword != "" {
		query = query.Where("check_in_point.title LIKE ?", "%"+keyword+"%")
	}
	if category != "" {
		query = query.Where("check_in_point.category = ?", category)
	}

	var allPithyPoints []CheckInInfoPithy
	err = query.Find(&allPithyPoints).Error
	if err != nil {
		return nil, err
	}
	// 计算hot排名，插入排名，而不是排序
	hotItems := make([]HotItem, 0)
	for _, point := range allPithyPoints {
		hotItems = append(hotItems, HotItem{
			SpotId:   point.SpotId,
			HotValue: point.HotValue,
		})
	}
	sort.Slice(hotItems, func(i, j int) bool {
		return hotItems[i].HotValue > hotItems[j].HotValue
	})

	hotRankMap := make(map[int64]int64)
	for i, item := range hotItems {
		hotRankMap[item.SpotId] = int64(i) + 1
	}
	for i, _ := range allPithyPoints {
		point := &allPithyPoints[i]
		if rank, exists := hotRankMap[point.SpotId]; exists {
			point.HotRank = rank
		} else {
			point.HotRank = 0
		}
	}

	pointIds := make([]int64, 0)
	for _, point := range allPithyPoints {
		pointIds = append(pointIds, point.SpotId)
	}
	product, err := getFirstProductBySpotIds(ctx, nu, pointIds, []int64{common.SpotAuditStatusApproved})
	if err != nil {
		return nil, err
	}
	sdlog.Infof("product: %v", product)
	productIds := make([]int64, 0)
	for _, p := range product {
		productIds = append(productIds, int64(p.ID))
	}

	productMap := make(map[int64]model.Product)
	for _, p := range product {
		productMap[p.CheckInPointID] = p
	}

	productImageMap, err := GetProductImageMapByProductIds(ctx, nu, productIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		return nil, err
	}

	productPriceMap := make(map[int64]ProductPrice)
	for _, p := range product {
		if p.AmountUsd > 0 {
			productPriceMap[int64(p.ID)] = ProductPrice{
				Price:    p.AmountUsd,
				Currency: "USDT",
			}
		} else if p.Amount > 0 {
			productPriceMap[int64(p.ID)] = ProductPrice{
				Price:    p.Amount,
				Currency: "FEC",
			}
		}
	}
	allPoints := make([]CheckInInfoPithyWithProduct, 0)
	for _, point := range allPithyPoints {
		checkInInfoPithyWithProduct := CheckInInfoPithyWithProduct{
			CheckInInfoPithy: point,
			MinPrice:         "",
			MinPriceCurrency: "",
			Products:         make([]ProductInfoWithImage, 0),
		}
		p, exists := productMap[point.SpotId]
		// var checkInInfoPithyWithProduct CheckInInfoPithyWithProduct
		if exists {
			var productWithImage ProductInfoWithImage
			productImages, exists := productImageMap[int64(p.ID)]
			if exists {
				productWithImage.HeadImg = productImages[0].ImageURL
				productWithImage.Images = productImages
			} else {
				productWithImage.HeadImg = ""
				productWithImage.Images = make([]model.ProductImage, 0)
			}
			productWithImage.Product = p
			if _, exists := productPriceMap[int64(p.ID)]; exists {
				if productPriceMap[int64(p.ID)].Price == 0 {
					checkInInfoPithyWithProduct.MinPrice = ""
					checkInInfoPithyWithProduct.MinPriceCurrency = ""
				} else {
					// 有小数的保留两位小数，没小数的保留整数
					if productPriceMap[point.SpotId].Price == math.Trunc(productPriceMap[point.SpotId].Price) {
						checkInInfoPithyWithProduct.MinPrice = fmt.Sprintf("%.0f", productPriceMap[int64(p.ID)].Price)
					} else {
						checkInInfoPithyWithProduct.MinPrice = fmt.Sprintf("%.2f", productPriceMap[int64(p.ID)].Price)
					}
					checkInInfoPithyWithProduct.MinPriceCurrency = productPriceMap[int64(p.ID)].Currency
				}
			} else {
				checkInInfoPithyWithProduct.MinPrice = ""
				checkInInfoPithyWithProduct.MinPriceCurrency = ""
			}
			checkInInfoPithyWithProduct.Products = []ProductInfoWithImage{productWithImage}
		} else {
			checkInInfoPithyWithProduct.Products = make([]ProductInfoWithImage, 0)
		}

		allPoints = append(allPoints, checkInInfoPithyWithProduct)
	}

	// 如果点数小于100，直接返回具体点信息
	if totalCount <= 100 {
		// 将点信息转换为网格格式
		for _, point := range allPoints {
			point.IsChina = false
			if point.Country == "中国" || point.Country == "中国大陆" || point.Country == "China" {
				point.IsChina = true
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
			cell := GridCell{
				// GridId:    fmt.Sprintf("%.6f,%.6f", point.Latitude, point.Longitude),
				Latitude:  point.Latitude,
				Longitude: point.Longitude,
				Count:     "1",
				Type:      "point",
				Points:    []CheckInInfoPithyWithProduct{point},
			}
			response.Grids = append(response.Grids, cell)
		}
	} else {
		// 点数大于100，使用网格化展示
		gridRange := calculateOptimalGridSize(swLat, swLng, neLat, neLng)

		// 使用已查询的点数据计算网格统计
		gridMap := make(map[string]GridInfo)

		// 统计每个网格的点数
		for _, point := range allPoints {
			gridKey := calculateGridKey(point, gridRange)
			gridMap = addPointToGrid(gridMap, point, gridKey)
		}

		// 构建网格响应
		for _, grid := range gridMap {

			countStr := fmt.Sprintf("%d", grid.Count)
			if grid.Count > 9 {
				grid.Points = grid.Points[:9]
			}

			typeValue := "grid"
			if len(grid.Points) == 1 {
				typeValue = ""
			}

			// 处理分类信息
			for key, point := range grid.Points {
				category, exists := memory.CategoryMemoryCache.GetCategoryList()[point.Category]
				if exists {
					grid.Points[key].CategoryImage = category.ImageURL
					if common.GetRequestLanguage() == common.LanguageZh {
						grid.Points[key].Category = category.LabelZh
					} else {
						grid.Points[key].Category = category.Label
					}
					grid.Points[key].CategoryEn = category.Type
				}
			}
			// 计算网格中心点
			avgLat, avgLng := calculateGridCenter(grid)
			cell := GridCell{
				Latitude:  avgLat,
				Longitude: avgLng,
				Count:     countStr,
				Type:      typeValue,
				Points:    grid.Points,
			}
			response.Grids = append(response.Grids, cell)
		}
	}

	return response, nil
}

type SpotListPithyResult struct {
	TotalCount int64                         `json:"total_count"` // 地标总数
	Points     []CheckInInfoPithyWithProduct `json:"points"`      // 地标列表
}

// SpotListPithy 获取地标列表
func SpotListPithy(ctx context.Context, nu *nucl.Nucleus, uid int64, swLat, swLng, neLat, neLng float64, isOwner int64, keyword, category string, page, pageSize int64) (*SpotListPithyResult, error) {
	// 首先查询可视区域内的总点数
	var totalCount int64
	countQuery := nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
		Where("latitude >= ? AND latitude <= ? AND longitude >= ? AND longitude <= ? AND status = 1 and is_active=1 and audit_status=1",
			swLat, neLat, swLng, neLng)

	// 添加条件查询
	if isOwner == 1 {
		countQuery = countQuery.Where("owner_id = ?", uid)
	}
	if keyword != "" {
		countQuery = countQuery.Where("title LIKE ?", "%"+keyword+"%")
	}
	if category != "" {
		countQuery = countQuery.Where("category = ?", category)
	}

	err := countQuery.Count(&totalCount).Error
	if err != nil {
		return nil, err
	}

	// 查询所有点信息
	query := nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
		Select("check_in_point.id as spot_id, check_in_point.latitude, check_in_point.longitude, check_in_point.title, check_in_point.category, check_in_point.owner_id, user.nickname as owner_username, user.avatar_url as owner_avatar, MIN(check_in_point_image.image_url) as landmark_image, check_in_point.location, check_in_point.like_count, check_in_point.checkin_count, check_in_point.checkin_like_count, check_in_point.checkin_comment_count, check_in_point.checkin_visit_count, check_in_point.checkin_count * 4 + check_in_point.checkin_like_count * 3 + check_in_point.checkin_comment_count * 2 + check_in_point.checkin_visit_count * 1 as hot_value").
		Joins("left join user on user.user_id = check_in_point.owner_id").
		Joins("left join check_in_point_image on check_in_point_image.check_in_point_id = check_in_point.id and check_in_point_image.status = 1").
		Where("check_in_point.status = 1 AND check_in_point.is_active = 1 AND check_in_point.audit_status = 1 AND check_in_point.latitude BETWEEN ? AND ? AND check_in_point.longitude BETWEEN ? AND ?",
			swLat, neLat, swLng, neLng).
		Group("check_in_point.id, check_in_point.latitude, check_in_point.longitude, check_in_point.title, check_in_point.category, check_in_point.owner_id, user.nickname, user.avatar_url, check_in_point.location")

	// 添加条件查询
	if isOwner == 1 {
		query = query.Where("check_in_point.owner_id = ?", uid)
	}
	if keyword != "" {
		query = query.Where("check_in_point.title LIKE ?", "%"+keyword+"%")
	}
	if category != "" {
		query = query.Where("check_in_point.category = ?", category)
	}

	var allPoints []CheckInInfoPithy
	err = query.Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&allPoints).Error
	if err != nil {
		return nil, err
	}

	// 计算hot排名
	hotItems := make([]HotItem, 0)
	for _, point := range allPoints {
		hotItems = append(hotItems, HotItem{
			SpotId:   point.SpotId,
			HotValue: point.HotValue,
		})
	}
	sort.Slice(hotItems, func(i, j int) bool {
		return hotItems[i].HotValue > hotItems[j].HotValue
	})

	hotRankMap := make(map[int64]int64)
	for i, item := range hotItems {
		hotRankMap[item.SpotId] = int64(i) + 1
	}
	for i, _ := range allPoints {
		point := &allPoints[i]
		if rank, exists := hotRankMap[point.SpotId]; exists {
			point.HotRank = rank
		} else {
			point.HotRank = 0
		}
	}
	spotIds := make([]int64, 0)
	for _, point := range allPoints {
		spotIds = append(spotIds, point.SpotId)
	}
	product, err := getFirstProductBySpotIds(ctx, nu, spotIds, []int64{common.SpotAuditStatusApproved})
	if err != nil {
		return nil, err
	}

	productIds := make([]int64, 0)
	for _, p := range product {
		productIds = append(productIds, int64(p.ID))
	}
	productImageMap, err := GetProductImageMapByProductIds(ctx, nu, productIds, []int{common.SpotAuditStatusApproved})
	if err != nil {
		return nil, err
	}
	productPriceMap := make(map[int64]ProductPrice)
	for _, p := range product {
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
	productWithImagesArrMapBySpotId := make(map[int64][]ProductInfoWithImage)
	for _, p := range product {

		pI := ProductInfoWithImage{
			Product: p,
			HeadImg: "",
			Images:  make([]model.ProductImage, 0),
		}
		if len(productImageMap[int64(p.ID)]) > 0 {
			pI.HeadImg = productImageMap[int64(p.ID)][0].ImageURL
			pI.Images = productImageMap[int64(p.ID)]
		}

		productWithImageArr, exists := productWithImagesArrMapBySpotId[int64(p.ID)]
		if exists {
			productWithImageArr = append(productWithImageArr, pI)
			productWithImagesArrMapBySpotId[p.CheckInPointID] = productWithImageArr
		} else {
			productWithImagesArrMapBySpotId[p.CheckInPointID] = []ProductInfoWithImage{pI}
		}

	}

	productMap := make(map[int64]model.Product)
	for _, p := range product {
		productMap[p.CheckInPointID] = p
	}

	var allPointsWithProduct []CheckInInfoPithyWithProduct
	for _, point := range allPoints {
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
		pointWithProduct := CheckInInfoPithyWithProduct{
			CheckInInfoPithy: point,
			MinPrice:         "",
			MinPriceCurrency: "",
			Products:         make([]ProductInfoWithImage, 0),
		}

		productWithImageArr, exists := productWithImagesArrMapBySpotId[point.SpotId]
		if exists {
			pointWithProduct.Products = productWithImageArr
		} else {
			pointWithProduct.Products = make([]ProductInfoWithImage, 0)
		}

		if _, exists := productPriceMap[point.SpotId]; exists {
			if productPriceMap[point.SpotId].Price == 0 {
				pointWithProduct.MinPrice = ""
				pointWithProduct.MinPriceCurrency = ""
			} else {
				if productPriceMap[point.SpotId].Price == math.Trunc(productPriceMap[point.SpotId].Price) {
					pointWithProduct.MinPrice = fmt.Sprintf("%.0f", productPriceMap[point.SpotId].Price)
				} else {
					pointWithProduct.MinPrice = fmt.Sprintf("%.2f", productPriceMap[point.SpotId].Price)
				}
				pointWithProduct.MinPriceCurrency = productPriceMap[point.SpotId].Currency
			}
		}

		allPointsWithProduct = append(allPointsWithProduct, pointWithProduct)
	}
	return &SpotListPithyResult{
		TotalCount: totalCount,
		Points:     allPointsWithProduct,
	}, nil
}

func getFirstProductBySpotIds(ctx context.Context, nu *nucl.Nucleus, spotIds []int64, audit_status []int64) ([]model.Product, error) {
	subQuery := nu.DB.
		Table("product as p2").
		Select("MIN(id)").
		Where("p2.check_in_point_id = p.check_in_point_id and p2.is_active = 1")
	if len(audit_status) != 0 {
		subQuery = subQuery.Where("p2.audit_status in (?)", audit_status)
	}
	product := make([]model.Product, 0)
	err := nu.DB.Table("product as p").
		Where("p.check_in_point_id IN ?", spotIds).
		Where("p.id = (?)", subQuery).
		Find(&product).Error
	if err != nil {
		return nil, err
	}
	sdlog.Info("product", "product", product)
	return product, nil
}
