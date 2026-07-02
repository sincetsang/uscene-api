package service

import (
	"TMA/pkg/cache/memory"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdparse"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"

	"gorm.io/gorm"
)

type CheckInInfoByDistance struct {
	TotalCount int64                                 `json:"total_count"` // 地标总数
	Points     []*db.CheckInPointWithOwnerByDistance `json:"points"`      // 地标列表
}

func SpotListByDistance(ctx context.Context, nu *nucl.Nucleus, uid int64, latitudeStr string, longitudeStr string, category string, page, page_size int64) (*CheckInInfoByDistance, error) {

	if latitudeStr == "" || longitudeStr == "" {
		return nil, common.ErrLocationLatLonErr
	}
	latitude := sdparse.Float64Def(latitudeStr, 0.0)
	longitude := sdparse.Float64Def(longitudeStr, 0.0)

	result := CheckInInfoByDistance{}

	checkPoints, count, errC := db.GetAllValidCheckPointsByLaLoByDistance(nu.DB.WithContext(ctx), latitude, longitude, uid, category, page, page_size)
	if errC != nil {
		return &result, errC
	}

	spotIds := make([]int64, len(checkPoints))
	for i, point := range checkPoints {
		spotIds[i] = point.ID
	}

	// 批量查询地标图片
	var pointImages []model.CheckInPointImage
	err := nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
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

	for _, point := range checkPoints {
		if point.LandmarkImage == "" {
			if images, exists := pointImagesMap[point.ID]; exists && len(images) > 0 {
				point.LandmarkImage = images[0].ImageURL
			}
		}
		// 设置分类图片
		if cat, exists := memory.CategoryMemoryCache.GetCategoryList()[point.Category]; exists {
			point.CategoryImage = cat.ImageURL
		}
	}
	result.Points = checkPoints
	result.TotalCount = count
	return &result, nil
}
