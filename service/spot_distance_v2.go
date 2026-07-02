package service

import (
	"TMA/pkg/cache/memory"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdparse"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"fmt"
	"math"

	"gorm.io/gorm"
)

type CheckInInfoWithProduct struct {
	*db.CheckInPointWithOwnerByDistance
	Products         []*model.Product `json:"products"` // 商品列表
	Category         string           `json:"category"`
	CategoryEn       string           `json:"category_en"`
	CategoryImage    string           `json:"category_image"`
	MinPrice         string           `json:"min_price"`
	MinPriceCurrency string           `json:"min_price_currency"`
}

type CheckInInfoWithProductByDistance struct {
	TotalCount int64                    `json:"total_count"` // 地标总数
	Points     []CheckInInfoWithProduct `json:"points"`      // 地标列表

}

func SpotListByDistanceV2(ctx context.Context, nu *nucl.Nucleus, uid int64, latitudeStr string, longitudeStr string, category string, page, page_size int64) (*CheckInInfoWithProductByDistance, error) {

	if latitudeStr == "" || longitudeStr == "" {
		return nil, common.ErrLocationLatLonErr
	}
	latitude := sdparse.Float64Def(latitudeStr, 0.0)
	longitude := sdparse.Float64Def(longitudeStr, 0.0)

	result := CheckInInfoWithProductByDistance{}

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

	products, err := getFirstProductBySpotIds(ctx, nu, spotIds, []int64{1})
	if err != nil {
		return nil, err
	}

	productsMap := make(map[int64]*model.Product)
	for _, product := range products {
		productsMap[product.CheckInPointID] = &product
	}

	productPriceMapBtSpotId := make(map[int64]ProductPrice)
	for _, p := range products {
		if p.AmountUsd > 0 {
			productPriceMapBtSpotId[int64(p.CheckInPointID)] = ProductPrice{
				Price:    p.AmountUsd,
				Currency: "USDT",
			}
		} else if p.Amount > 0 {
			productPriceMapBtSpotId[int64(p.CheckInPointID)] = ProductPrice{
				Price:    p.Amount,
				Currency: "FEC",
			}
		}
	}
	var resultPoints []CheckInInfoWithProduct
	for _, point := range checkPoints {
		pointWithProduct := CheckInInfoWithProduct{
			CheckInPointWithOwnerByDistance: point,
			Products:                        make([]*model.Product, 0),
			MinPrice:                        "",
			MinPriceCurrency:                "",
		}
		category, exists := memory.CategoryMemoryCache.GetCategoryList()[point.Category]
		if exists {
			point.CategoryImage = category.ImageURL
			if common.GetRequestLanguage() == common.LanguageZh {
				pointWithProduct.Category = category.LabelZh
			} else {
				pointWithProduct.Category = category.Label
			}
			pointWithProduct.CategoryEn = category.Type
			pointWithProduct.CategoryImage = category.ImageURL
		}
		if product, exists := productsMap[point.ID]; exists {
			pointWithProduct.Products = []*model.Product{product}
		}
		if productPrice, exists := productPriceMapBtSpotId[point.ID]; exists {
			if productPrice.Price == math.Trunc(productPrice.Price) {
				pointWithProduct.MinPrice = fmt.Sprintf("%.0f", productPrice.Price)
			} else {
				pointWithProduct.MinPrice = fmt.Sprintf("%.2f", productPrice.Price)
			}
			pointWithProduct.MinPriceCurrency = productPrice.Currency
		}
		resultPoints = append(resultPoints, pointWithProduct)
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
	result.Points = resultPoints
	result.TotalCount = count

	return &result, nil
}
