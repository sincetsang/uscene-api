package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"github.com/sulink/ueapisdk/utils/constants"

	"gorm.io/gorm"
)

// CreateOrderRequest 创建订单请求参数
type CreateOrderRequest struct {
	ProductID int64  `json:"product_id"` // 产品ID
	PayType   string `json:"pay_type"`   // 支付方式

}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderID     int64     `json:"order_id"`     // 订单ID
	ProductID   int64     `json:"product_id"`   // 产品ID
	ProductName string    `json:"product_name"` // 产品名称
	Amount      float64   `json:"amount"`       // 订单金额
	CodeNum     int32     `json:"code_num"`     // 核销码数量
	PayType     string    `json:"pay_type"`     // 支付方式
	CreatedAt   time.Time `json:"created_at"`
	ExpiredAt   time.Time `json:"expired_at"` // 订单过期时间
}

// PayOrderResponse 支付订单响应
type PayOrderResponse struct {
	OrderID string  `json:"order_id"` // 订单ID
	Paycode string  `json:"paycode"`  // 支付码
	Amount  float64 `json:"amount"`
	PayType int64   `json:"pay_type"`
	Unit    int64   `json:"unit"`
}

// OrderListItem 订单列表项
type OrderListItem struct {
	OrderID      int64     `json:"order_id"`                     // 订单ID
	ProductID    int64     `json:"product_id"`                   // 产品ID
	ProductName  string    `json:"product_name"`                 // 产品名称
	ProductNum   int32     `json:"product_num"`                  // 产品数量
	ProductPrice float64   `json:"product_price"`                // 产品价格
	Amount       float64   `json:"amount"`                       // 订单金额
	CodeNum      int32     `json:"code_num"`                     // 核销码数量
	Status       string    `json:"status"`                       // 订单状态
	PayType      string    `json:"pay_type"`                     // 支付方式
	Unit         string    `json:"unit"`                         // 货币
	CreatedAt    string    `json:"created_at"`                   // 创建时间
	CAt          time.Time `json:"c_at"`                         // 创建时间v2
	UAt          time.Time `json:"u_at"`                         // 更新时间v2
	ExpiredAt    time.Time `json:"expired_at"`                   // 订单过期时间
	UpdatedAt    string    `json:"updated_at"`                   // 更新时间
	HeadImg      string    `gorm:"column:image" json:"head_img"` // 产品头图
	Images       []string  `json:"images"`                       // 产品图片列表
}

// OrderListResponse 订单列表响应
type OrderListResponse struct {
	Total int64           `json:"total"` // 总数
	Items []OrderListItem `json:"items"` // 订单列表
}

// GetOrderPayStatusResponse 获取订单支付状态响应
// 返回订单ID和状态
// status 取值：unpaid, paid, cancel
type GetOrderPayStatusResponse struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}

// createOrder 创建产品订单
func createOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始创建产品订单")

	// 解析请求参数
	req := CreateOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 参数验证
	if req.ProductID <= 0 {
		sdlog.Error("产品ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询产品信息
	var product model.Product
	err := ec.Nu.DB.Model(&model.Product{}).
		Where("id = ?", req.ProductID).
		First(&product).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("产品不存在: %d", req.ProductID)
			return webapi.Error(common.ErrNotFoundProduct).Render(ec)
		}
		sdlog.Errorf("查询产品信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建订单
	orderID, _ := strconv.ParseInt(sdrand.GenerateSecureString(10, true, false, false), 10, 64)
	order := &model.VerificationCodeOrder{
		OrderID:     orderID,
		UserID:      ec.AuthData.User.ID,
		ProductID:   int64(product.ID),
		ProductName: product.Name,
		CodeNum:     product.CheckCodeNum,
		Amount:      decimal.NewFromFloat(product.Amount).InexactFloat64(),
		PayType:     "FEC",
		Unit:        "USD",
		Status:      common.VerificationCodeOrderStatusUnpaid, // 0: 待支付
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = ec.Nu.DB.Create(order).Error
	if err != nil {
		sdlog.Errorf("创建订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// // 启动异步任务，15分钟后检查订单状态并取消未支付订单
	// go func(orderID int64) {
	// 	time.Sleep(15 * time.Minute)
	// 	var order model.VerificationCodeOrder
	// 	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
	// 		Where("order_id = ? AND status = ?", orderID, common.VerificationCodeOrderStatusUnpaid).
	// 		First(&order).Error
	// 	if err == nil {
	// 		// 订单存在且未支付，则取消订单
	// 		err = ec.Nu.DB.Model(&order).Update("status", common.VerificationCodeOrderStatusCancel).Error
	// 		if err != nil {
	// 			sdlog.Errorf("自动取消订单失败: %v", err)
	// 		} else {
	// 			sdlog.Infof("订单 %d 超时未支付，已自动取消", orderID)
	// 		}
	// 	}
	// }(order.ID)

	sdlog.Infof("创建订单成功，订单ID: %d", order.ID)
	return webapi.OK(CreateOrderResponse{
		OrderID:     order.OrderID,
		ProductID:   order.ProductID,
		ProductName: order.ProductName,
		Amount:      decimal.NewFromFloat(order.Amount).InexactFloat64(),
		CodeNum:     order.CodeNum,
		PayType:     order.PayType,
		CreatedAt:   order.CreatedAt.Truncate(time.Second),
		ExpiredAt:   order.CreatedAt.Truncate(time.Second).Add(15 * time.Minute),
	}).Render(ec)
}

// getPayInfo 获取支付信息
func getPayInfo(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取支付信息")

	// 获取订单ID
	orderID := ec.QueryParams().Get("order_id")
	if orderID == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.VerificationCodeOrder
	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", orderID).
		Where("user_id = ?", ec.AuthData.User.ID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrParam).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证订单状态
	if order.Status != common.VerificationCodeOrderStatusUnpaid {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 验证订单所属
	if order.UserID != ec.AuthData.User.ID {
		sdlog.Error("无权访问此订单")
		return webapi.Error(common.ErrUnPower).Render(ec)
	}
	if order.UeOrderID != "" {
		sdlog.Infof("订单已存在，支付信息为，订单ID: %s", orderID)
		return webapi.OK(PayOrderResponse{
			OrderID: strconv.FormatInt(order.OrderID, 10),
			Paycode: ec.Nu.Config.UE.PayCode,
			Amount:  order.Amount,
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
			Amount:  decimal.NewFromFloat(order.Amount).InexactFloat64(),
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

	sdlog.Infof("创建支付订单成功: %+v", resp)
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
	return webapi.OK(PayOrderResponse{
		OrderID: ueReq.Orderno,
		Paycode: ueReq.Paycode,
		Amount:  decimal.NewFromFloat(order.Amount).InexactFloat64(),
		PayType: 25,
		Unit:    16,
	}).Render(ec)
}

// getOrderList 获取订单列表
func getOrderList(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取订单列表")

	status := ec.QueryParams().Get("status")
	validStatuses := map[string]bool{
		common.VerificationCodeOrderStatusUnpaid: true,
		common.VerificationCodeOrderStatusPaid:   true,
		common.VerificationCodeOrderStatusCancel: true,
	}

	if !validStatuses[status] {
		status = ""
	}
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
		sdlog.Error("页码格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeNum, err := strconv.ParseInt(pageSize, 10, 64)
	if err != nil {
		sdlog.Error("每页数量格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单总数
	var total int64
	if status == "" {
		err = ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Count(&total).Error
	} else {
		err = ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Where("status = ?", status).
			Count(&total).Error
	}
	if err != nil {
		sdlog.Errorf("查询订单总数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 查询订单列表
	var orders []model.VerificationCodeOrder
	if status == "" {
		err = ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Order("created_at DESC").
			Offset(int((pageNum - 1) * pageSizeNum)).
			Limit(int(pageSizeNum)).
			Find(&orders).Error
	} else {
		err = ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Where("status = ?", status).
			Order("created_at DESC").
			Offset(int((pageNum - 1) * pageSizeNum)).
			Limit(int(pageSizeNum)).
			Find(&orders).Error
	}

	if err != nil {
		sdlog.Errorf("查询订单列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取所有产品ID
	var productIDs []int64
	for _, order := range orders {
		productIDs = append(productIDs, order.ProductID)
	}

	// 查询产品图片
	var productImages []model.ProductImage
	err = ec.Nu.DB.Model(&model.ProductImage{}).
		Where("product_id IN ? AND status = 1", productIDs).
		Order("sort_index ASC").
		Find(&productImages).Error
	if err != nil {
		sdlog.Errorf("查询产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建产品ID到图片URL的映射
	productImageMap := make(map[int64][]string)
	for _, image := range productImages {
		productImageMap[image.ProductID] = append(productImageMap[image.ProductID], image.ImageURL)
	}

	// 查询产品信息
	var products []model.Product
	err = ec.Nu.DB.Model(&model.Product{}).
		Where("id IN ?", productIDs).
		Find(&products).Error
	if err != nil {
		sdlog.Errorf("查询产品信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建产品ID到产品信息的映射
	productMap := make(map[int64]model.Product)
	for _, product := range products {
		productMap[int64(product.ID)] = product
	}

	// 构建响应数据
	items := make([]OrderListItem, 0, len(orders))
	for _, order := range orders {
		items = append(items, OrderListItem{
			OrderID:      order.OrderID,
			ProductID:    order.ProductID,
			ProductName:  order.ProductName,
			ProductNum:   order.ProductNum,
			ProductPrice: decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
			Amount:       decimal.NewFromFloat(order.Amount).InexactFloat64(),
			CodeNum:      order.CodeNum,
			Status:       order.Status,
			PayType:      order.PayType,
			Unit:         order.Unit,
			CreatedAt:    order.CreatedAt.Format("2006-01-02 15:04:05"),
			CAt:          order.CreatedAt,
			UAt:          order.UpdatedAt,
			UpdatedAt:    order.UpdatedAt.Format("2006-01-02 15:04:05"),
			ExpiredAt:    order.CreatedAt.Add(15 * time.Minute),
			HeadImg:      productMap[order.ProductID].Image,
			Images:       productImageMap[order.ProductID],
		})
	}

	sdlog.Infof("获取订单列表成功，总数: %d", total)
	return webapi.OK(OrderListResponse{
		Total: total,
		Items: items,
	}).Render(ec)
}

// getOrderPayStatus 获取订单支付状态
func getOrderPayStatus(ec *middleware.AppRequestContext) error {
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

	var order model.VerificationCodeOrder
	err = ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
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

type CancelOrderRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

// CancelOrder 取消订单
func CancelOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始取消订单")

	// 获取订单ID
	req := CancelOrderRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderID := req.OrderID
	if orderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单
	var order model.VerificationCodeOrder
	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
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
	if order.Status != common.VerificationCodeOrderStatusUnpaid {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 更新订单状态为已取消
	err = ec.Nu.DB.Model(&order).Update("status", common.VerificationCodeOrderStatusCancel).Error
	if err != nil {
		sdlog.Errorf("取消订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 取消成功", orderID)
	return webapi.OK(true).Render(ec)
}
