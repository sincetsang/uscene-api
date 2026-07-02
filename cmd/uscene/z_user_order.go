package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdparse"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"strings"
)

// 分页
// 根据状态查询 status 有以下几种: pending_payment, pending_shipment, pending_receipt, completed, cancelled, refund_in_progress, refunded
// 如果 status 为空，则查询所有状态
// 如果查询待评论，则 status 为 completed
// spot image
// product image 也需要查询

type SpotWithImage struct {
	model.CheckInPoint
	Image string `json:"image" gorm:"-"`
}

type ProductWithImage struct {
	model.Product
	HeadImg string               `json:"head_img" gorm:"-"`
	Images  []model.ProductImage `json:"images" gorm:"-"`
}

type ProductOrderWithSpot struct {
	Spot    SpotWithImage      `json:"spot"`
	Product ProductWithImage   `json:"product"`
	Order   model.ProductOrder `json:"order"`
}
type UserProductBoughtRes struct {
	Total    int64                  `json:"total"`
	Items    []ProductOrderWithSpot `json:"items"`
	Page     int64                  `json:"page"`
	PageSize int64                  `json:"page_size"`
}

func getUserProductBought(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	status := ec.QueryParams().Get("status")
	// 传入的字符串用逗号分割，转换为数组
	statusArray := []string{}
	if status != "" {
		statusArray = strings.Split(status, ",")
	}
	// 待评论
	pendingComment := sdparse.BoolDef(ec.QueryParams().Get("pending_comment"), false)
	// 退款中
	refunding := sdparse.BoolDef(ec.QueryParams().Get("pending_refund"), false)
	basequery := ec.Nu.DB.Model(&model.ProductOrder{}).Where("user_id = ? and is_deleted = ?", userId, false)
	if pendingComment {
		statusArray = []string{common.ProductOrderStatusCompleted}
		basequery = basequery.Where("is_commented = ?", false).Where("status = ?", common.ProductOrderStatusCompleted)
	} else if refunding {
		statusArray = []string{common.ProductRefundStatusRefunding}
		basequery = basequery.Where("refund_status IN (?)", statusArray)
	} else {
		if len(statusArray) > 0 {
			basequery = basequery.Where("status IN (?)", statusArray)
		}
	}

	var total int64
	err := basequery.Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if total == 0 {
		return webapi.OK(UserProductBoughtRes{
			Total:    total,
			Items:    []ProductOrderWithSpot{},
			Page:     pageInt,
			PageSize: pageSizeInt,
		}).Render(ec)
	}

	var orders []model.ProductOrder

	err = basequery.Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&orders).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	spotIds := []int64{}
	for _, order := range orders {
		spotIds = append(spotIds, order.SpotID)
	}
	spots := []model.CheckInPoint{}
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id IN (?)", spotIds).Find(&spots).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	spotMap := make(map[int64]model.CheckInPoint)
	for _, spot := range spots {
		spotMap[spot.ID] = spot
	}

	productIds := []int64{}
	for _, order := range orders {
		productIds = append(productIds, order.ProductID)
	}
	products := []model.Product{}
	err = ec.Nu.DB.Model(&model.Product{}).Where("id IN (?)", productIds).Find(&products).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	productMap := make(map[int64]model.Product)
	for _, product := range products {
		productMap[int64(product.ID)] = product
	}

	// 查询地标图片
	spotImages := []model.CheckInPointImage{}
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).Where("check_in_point_id IN (?) and status = 1", spotIds).Find(&spotImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	spotImageMap := make(map[int64][]model.CheckInPointImage)
	for _, image := range spotImages {
		spotImageMap[image.CheckInPointID] = append(spotImageMap[image.CheckInPointID], image)
	}

	// 查询产品图片
	productImages := []model.ProductImage{}
	err = ec.Nu.DB.Model(&model.ProductImage{}).Where("product_id IN (?) and status = 1", productIds).Find(&productImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	productImageMap := make(map[int64][]model.ProductImage)
	for _, image := range productImages {
		productImageMap[image.ProductID] = append(productImageMap[image.ProductID], image)
	}

	items := make([]ProductOrderWithSpot, 0, len(orders))
	for _, order := range orders {
		item := ProductOrderWithSpot{
			Order: order,
		}
		spot, ok := spotMap[order.SpotID]
		if ok {
			item.Spot = SpotWithImage{
				CheckInPoint: spot,
			}
			if spotImages, ok := spotImageMap[order.SpotID]; ok {
				item.Spot.Image = spotImages[0].ImageURL
			}
		}
		product, ok := productMap[order.ProductID]
		if ok {
			item.Product = ProductWithImage{
				Product: product,
			}
			if productImages, ok := productImageMap[order.ProductID]; ok {
				item.Product.HeadImg = productImages[0].ImageURL
				item.Product.Images = productImages
			}
		}
		items = append(items, item)
	}

	return webapi.OK(UserProductBoughtRes{
		Total:    total,
		Items:    items,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}).Render(ec)
}

func getUserProductSold(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId <= 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	status := ec.QueryParams().Get("status")
	// 传入的字符串用逗号分割，转换为数组
	statusArray := []string{}
	if status != "" {
		statusArray = strings.Split(status, ",")
	}
	// 待评论
	pendingComment := sdparse.BoolDef(ec.QueryParams().Get("pending_comment"), false)
	// 退款中
	refunding := sdparse.BoolDef(ec.QueryParams().Get("pending_refund"), false)
	basequery := ec.Nu.DB.Model(&model.ProductOrder{}).Where("seller_id = ? and is_deleted = ?", userId, false)
	if pendingComment {
		statusArray = []string{common.ProductOrderStatusCompleted}
		basequery = basequery.Where("is_commented = ?", false).Where("status = ?", common.ProductOrderStatusCompleted)
	} else if refunding {
		statusArray = []string{common.ProductRefundStatusRefunding}
		basequery = basequery.Where("refund_status IN (?)", statusArray)
	} else {
		if len(statusArray) > 0 {
			basequery = basequery.Where("status IN (?)", statusArray)
		}
	}
	var total int64
	err := basequery.Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if total == 0 {
		return webapi.OK(UserProductBoughtRes{
			Total:    total,
			Items:    []ProductOrderWithSpot{},
			Page:     pageInt,
			PageSize: pageSizeInt,
		}).Render(ec)
	}

	var orders []model.ProductOrder

	err = basequery.Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&orders).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	spotIds := []int64{}
	for _, order := range orders {
		spotIds = append(spotIds, order.SpotID)
	}
	spots := []model.CheckInPoint{}
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id IN (?)", spotIds).Find(&spots).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	spotMap := make(map[int64]model.CheckInPoint)
	for _, spot := range spots {
		spotMap[spot.ID] = spot
	}

	productIds := []int64{}
	for _, order := range orders {
		productIds = append(productIds, order.ProductID)
	}
	products := []model.Product{}
	err = ec.Nu.DB.Model(&model.Product{}).Where("id IN (?)", productIds).Find(&products).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	productMap := make(map[int64]model.Product)
	for _, product := range products {
		productMap[int64(product.ID)] = product
	}

	// 查询地标图片
	spotImages := []model.CheckInPointImage{}
	err = ec.Nu.DB.Model(&model.CheckInPointImage{}).Where("check_in_point_id IN (?) and status = 1", spotIds).Find(&spotImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	spotImageMap := make(map[int64][]model.CheckInPointImage)
	for _, image := range spotImages {
		spotImageMap[image.CheckInPointID] = append(spotImageMap[image.CheckInPointID], image)
	}

	// 查询产品图片
	productImages := []model.ProductImage{}
	err = ec.Nu.DB.Model(&model.ProductImage{}).Where("product_id IN (?) and status = 1", productIds).Find(&productImages).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	productImageMap := make(map[int64][]model.ProductImage)
	for _, image := range productImages {
		productImageMap[image.ProductID] = append(productImageMap[image.ProductID], image)
	}

	items := make([]ProductOrderWithSpot, 0, len(orders))
	for _, order := range orders {
		item := ProductOrderWithSpot{
			Order: order,
		}
		spot, ok := spotMap[order.SpotID]
		if ok {
			item.Spot = SpotWithImage{
				CheckInPoint: spot,
			}
			if spotImages, ok := spotImageMap[order.SpotID]; ok {
				item.Spot.Image = spotImages[0].ImageURL
			}
		}
		product, ok := productMap[order.ProductID]
		if ok {
			item.Product = ProductWithImage{
				Product: product,
			}
			if productImages, ok := productImageMap[order.ProductID]; ok {
				item.Product.HeadImg = productImages[0].ImageURL
				item.Product.Images = productImages
			}
		}
		items = append(items, item)
	}

	return webapi.OK(UserProductBoughtRes{
		Total:    total,
		Items:    items,
		Page:     pageInt,
		PageSize: pageSizeInt,
	}).Render(ec)
}
