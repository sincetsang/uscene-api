package service

import (
	"TMA/pkg/nucl"
	"TMA/service/db/model"
	"context"

	"gorm.io/gorm"
)

func GetUserProduct(ctx context.Context, nu *nucl.Nucleus, uid int64, page, pageSize int64) ([]model.Product, error) {
	query := nu.DB.WithContext(ctx).Model(&model.Product{}).
		Joins("left join check_in_point on check_in_point.id = product.check_in_point_id").
		Where("check_in_point.owner_id = ?", uid).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize))

	var products []model.Product
	err := query.Find(&products).Error
	if err != nil {
		return nil, err
	}

	return products, nil
}

func GetUserFavoritesProductCount(ctx context.Context, nu *nucl.Nucleus, uid int64) (int64, error) {
	var total int64
	err := nu.DB.WithContext(ctx).Model(&model.Product{}).
		Joins("left join product_favorite on product_favorite.product_id = product.id").
		Where("product_favorite.user_id = ? and product.audit_status = ?", uid, 1).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func GetUserFavoritesProduct(ctx context.Context, nu *nucl.Nucleus, uid int64, page, pageSize int64) ([]model.Product, error) {
	query := nu.DB.WithContext(ctx).Model(&model.Product{}).
		Select("product.*").
		Joins("left join product_favorite on product_favorite.product_id = product.id").
		Where("product_favorite.user_id = ? and product.audit_status = ?", uid, 1).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize))

	var products []model.Product
	err := query.Find(&products).Error
	if err != nil {
		return nil, err
	}

	return products, nil
}

// 添加用户产品浏览历史
// 如果存在则删除之前记录，不存在则添加新纪录
func AddUserProductVisitHistory(ctx context.Context, nu *nucl.Nucleus, uid int64, productId int64) error {
	query := nu.DB.WithContext(ctx).Model(&model.ProductVisitHistory{}).
		Where("user_id = ? and product_id = ?", uid, productId).Delete(&model.ProductVisitHistory{})
	if query.Error != nil && query.Error != gorm.ErrRecordNotFound {
		return query.Error
	}

	query = nu.DB.WithContext(ctx).Model(&model.ProductVisitHistory{}).Create(&model.ProductVisitHistory{
		UserID:    uid,
		ProductID: productId,
	})
	if query.Error != nil {
		return query.Error
	}

	return nil
}

func GetUserProductVisitHistoryCount(ctx context.Context, nu *nucl.Nucleus, uid int64) (int64, error) {
	var total int64
	err := nu.DB.WithContext(ctx).Model(&model.ProductVisitHistory{}).
		Where("user_id = ?", uid).Count(&total).Error
	return total, err
}

func GetUserProductVisitHistory(ctx context.Context, nu *nucl.Nucleus, uid int64, page, pageSize int64) ([]model.Product, error) {
	query := nu.DB.WithContext(ctx).Model(&model.Product{}).
		Joins("left join product_visit_history on product.id = product_visit_history.product_id").
		Where("product_visit_history.user_id = ?", uid).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize))

	var products []model.Product
	err := query.Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

func GetProductImagesByProductIds(ctx context.Context, nu *nucl.Nucleus, productIds []int64, status []int) ([]model.ProductImage, error) {
	query := nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
		Where("product_id IN ?", productIds)
	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}
	query = query.Order("product_id").Order("sort_index")
	var productImages []model.ProductImage
	err := query.Find(&productImages).Error
	if err != nil {
		return nil, err
	}
	return productImages, nil
}

func GetProductImageMapByProductIds(ctx context.Context, nu *nucl.Nucleus, productIds []int64, status []int) (map[int64][]model.ProductImage, error) {
	query := nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
		Where("product_id IN ?", productIds)
	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}
	query = query.Order("product_id").Order("sort_index")
	var productImages []model.ProductImage
	err := query.Find(&productImages).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	productImageMap := make(map[int64][]model.ProductImage)
	for _, productImage := range productImages {
		productImageMap[productImage.ProductID] = append(productImageMap[productImage.ProductID], productImage)
	}
	return productImageMap, nil
}

func CheckUserIsProductOwner(ctx context.Context, nu *nucl.Nucleus, uid int64, productId int64) (bool, error) {
	query := nu.DB.WithContext(ctx).Model(&model.Product{}).
		Joins("left join check_in_point on check_in_point.id = product.check_in_point_id").
		Where("product.id = ? and check_in_point.owner_id = ?", productId, uid).
		First(&model.Product{})
	if query.Error != nil {
		if query.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, query.Error
	}
	return true, nil
}

func GetProductTagList(ctx context.Context, nu *nucl.Nucleus) ([]model.ProductTagConfig, error) {
	query := nu.DB.WithContext(ctx).Model(&model.ProductTagConfig{}).Order("sort_index")
	var productTagList []model.ProductTagConfig
	err := query.Find(&productTagList).Error
	if err != nil {
		return nil, err
	}
	return productTagList, nil
}

func GetProductTagMap(ctx context.Context, nu *nucl.Nucleus) (map[string]model.ProductTagConfig, error) {
	productTagList, err := GetProductTagList(ctx, nu)
	if err != nil {
		return nil, err
	}
	productTagMap := make(map[string]model.ProductTagConfig)
	for _, productTag := range productTagList {
		productTagMap[productTag.Mark] = productTag
	}
	return productTagMap, nil
}
