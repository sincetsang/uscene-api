package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// CreateOrderRequestV2 创建订单请求参数
type CreateOrderRequestV2 struct {
	ProductID int64  `json:"product_id"` // 产品ID
	PayType   string `json:"pay_type"`   // 支付方式
	Currency  string `json:"currency"`   // 货币
	Num       int32  `json:"num"`        // 数量
}

// CreateOrderResponseV2 创建订单响应
type CreateOrderResponseV2 struct {
	OrderID      int64     `json:"order_id"`      // 订单ID
	ProductID    int64     `json:"product_id"`    // 产品ID
	ProductName  string    `json:"product_name"`  // 产品名称
	ProductNum   int32     `json:"product_num"`   // 产品数量
	ProductPrice float64   `json:"product_price"` // 产品价格
	Amount       float64   `json:"amount"`        // 订单金额
	CodeNum      int32     `json:"code_num"`      // 核销码数量
	PayType      string    `json:"pay_type"`      // 支付方式
	Currency     string    `json:"currency"`      // 货币
	CreatedAt    time.Time `json:"created_at"`
	ExpiredAt    time.Time `json:"expired_at"` // 订单过期时间
}

// createOrderV2 创建产品订单
func createOrderV2(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始创建产品订单")

	// 解析请求参数
	req := CreateOrderRequestV2{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Error("解析请求参数失败")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 参数验证
	if req.ProductID <= 0 {
		sdlog.Error("产品ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Num <= 0 {
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
	productAmount := product.Amount * float64(req.Num)
	productPrice := product.Amount
	if req.PayType != "FEC" {
		productAmount = product.AmountUsd * float64(req.Num)
		productPrice = product.AmountUsd
	}
	orderID, _ := strconv.ParseInt(sdrand.GenerateSecureString(10, true, false, false), 10, 64)
	order := &model.VerificationCodeOrder{
		OrderID:      orderID,
		UserID:       ec.AuthData.User.ID,
		ProductID:    int64(product.ID),
		ProductName:  product.Name,
		CodeNum:      product.CheckCodeNum * int32(req.Num),
		ProductNum:   req.Num,
		ProductPrice: productPrice,
		Amount:       productAmount,
		PayType:      req.PayType,
		Unit:         req.Currency,
		Status:       common.VerificationCodeOrderStatusUnpaid, // 0: 待支付
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = ec.Nu.DB.Create(order).Error
	if err != nil {
		sdlog.Errorf("创建订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("创建订单成功，订单ID: %d", order.ID)
	return webapi.OK(CreateOrderResponseV2{
		OrderID:      order.OrderID,
		ProductID:    order.ProductID,
		ProductName:  order.ProductName,
		Amount:       order.Amount,
		ProductNum:   order.ProductNum,
		ProductPrice: order.ProductPrice,
		CodeNum:      order.CodeNum,
		PayType:      order.PayType,
		Currency:     order.Unit,
		CreatedAt:    order.CreatedAt.Truncate(time.Second),
		ExpiredAt:    order.CreatedAt.Truncate(time.Second).Add(15 * time.Minute),
	}).Render(ec)
}

type CancelOrderRequestV2 struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

// CancelOrderV2 取消订单
func CancelOrderV2(ec *middleware.AppRequestContext) error {
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

	// 若存在支付订单，则取消支付订单
	var payOrder model.PaymentOrder
	err = ec.Nu.DB.Model(&model.PaymentOrder{}).
		Where("order_id = ?", orderID).
		First(&payOrder).Error
	if err == nil {
		// 更新支付订单状态为已取消
		err = ec.Nu.DB.Model(&payOrder).Update("status", common.OrderStatusCancel).Error
		if err != nil {
			sdlog.Errorf("取消支付订单失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	sdlog.Infof("订单 %s 取消成功", orderID)
	return webapi.OK(true).Render(ec)
}

// OrderDetailResponseV2 订单详情响应V2
type OrderDetailResponseV2 struct {
	OrderID      int64     `json:"order_id"`      // 订单ID
	ProductID    int64     `json:"product_id"`    // 产品ID
	ProductName  string    `json:"product_name"`  // 产品名称
	ProductNum   int32     `json:"product_num"`   // 产品数量
	ProductPrice float64   `json:"product_price"` // 产品价格
	Amount       float64   `json:"amount"`        // 订单金额
	CodeNum      int32     `json:"code_num"`      // 核销码数量
	Status       string    `json:"status"`        // 订单状态
	PayType      string    `json:"pay_type"`      // 支付方式
	Currency     string    `json:"currency"`      // 货币
	CreatedAt    time.Time `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`    // 更新时间
	ExpiredAt    time.Time `json:"expired_at"`    // 订单过期时间
	UeOrderID    string    `json:"ue_order_id"`   // UE订单ID

	// 产品详细信息
	Product struct {
		ID             int32    `json:"id"`              // 产品ID
		Name           string   `json:"name"`            // 产品名称
		Description    string   `json:"description"`     // 产品描述
		Amount         float64  `json:"amount"`          // 产品价格
		CheckCodeNum   int32    `json:"check_code_num"`  // 检查码数量
		Level          int32    `json:"level"`           // 产品等级
		LevelTag       string   `json:"level_tag"`       // 等级标签
		Category       string   `json:"category"`        // 产品分类
		Image          string   `json:"image"`           // 产品头图
		OriginalPrice  float64  `json:"original_price"`  // 原价
		AmountCurrency string   `json:"amount_currency"` // 货币类型
		AmountUsd      float64  `json:"amount_usd"`      // USD价格
		Images         []string `json:"images"`          // 产品图片列表
	} `json:"product"`

	// 核销码信息
	VerificationCodes []struct {
		ID        int64     `json:"id"`         // 核销码ID
		Code      string    `json:"code"`       // 核销码
		Status    string    `json:"status"`     // 状态
		UseUID    int64     `json:"use_uid"`    // 使用用户ID
		UseTime   time.Time `json:"use_time"`   // 使用时间
		CreatedAt time.Time `json:"created_at"` // 创建时间
	} `json:"verification_codes"`
}

// getOrderDetailV2 获取订单详情V2
func getOrderDetailV2(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取订单详情V2")

	// 获取订单ID
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

	// 查询订单信息
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

	// 查询产品详细信息
	var product model.Product
	err = ec.Nu.DB.Model(&model.Product{}).
		Where("id = ?", order.ProductID).
		First(&product).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("产品不存在: %d", order.ProductID)
			return webapi.Error(common.ErrNotFoundProduct).Render(ec)
		}
		sdlog.Errorf("查询产品信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询产品图片
	var productImages []model.ProductImage
	err = ec.Nu.DB.Model(&model.ProductImage{}).
		Where("product_id = ? AND status = 1", order.ProductID).
		Order("sort_index ASC").
		Find(&productImages).Error
	if err != nil {
		sdlog.Errorf("查询产品图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建产品图片URL列表
	var imageURLs []string
	for _, image := range productImages {
		imageURLs = append(imageURLs, image.ImageURL)
	}

	// 查询核销码信息
	var verificationCodes []model.VerificationCode
	err = ec.Nu.DB.Model(&model.VerificationCode{}).
		Where("order_id = ?", order.OrderID).
		Order("created_at ASC").
		Find(&verificationCodes).Error
	if err != nil {
		sdlog.Errorf("查询核销码失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建核销码信息
	var codeInfos []struct {
		ID        int64     `json:"id"`
		Code      string    `json:"code"`
		Status    string    `json:"status"`
		UseUID    int64     `json:"use_uid"`
		UseTime   time.Time `json:"use_time"`
		CreatedAt time.Time `json:"created_at"`
	}

	for _, code := range verificationCodes {
		codeInfos = append(codeInfos, struct {
			ID        int64     `json:"id"`
			Code      string    `json:"code"`
			Status    string    `json:"status"`
			UseUID    int64     `json:"use_uid"`
			UseTime   time.Time `json:"use_time"`
			CreatedAt time.Time `json:"created_at"`
		}{
			ID:        code.ID,
			Code:      code.Code,
			Status:    code.Status,
			UseUID:    code.UseUID,
			UseTime:   code.UseTime,
			CreatedAt: code.CreatedAt,
		})
	}

	// 构建响应数据
	response := OrderDetailResponseV2{
		OrderID:           order.OrderID,
		ProductID:         order.ProductID,
		ProductName:       order.ProductName,
		ProductNum:        order.ProductNum,
		ProductPrice:      order.ProductPrice,
		Amount:            order.Amount,
		CodeNum:           order.CodeNum,
		Status:            order.Status,
		PayType:           order.PayType,
		Currency:          order.Unit,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
		ExpiredAt:         order.CreatedAt.Add(15 * time.Minute),
		UeOrderID:         order.UeOrderID,
		VerificationCodes: codeInfos,
	}

	// 填充产品信息
	response.Product.ID = product.ID
	response.Product.Name = product.Name
	response.Product.Description = product.Description
	response.Product.Amount = product.Amount
	response.Product.CheckCodeNum = product.CheckCodeNum
	response.Product.Level = product.Level
	response.Product.LevelTag = product.LevelTag
	response.Product.Category = product.Category
	response.Product.Image = product.Image
	response.Product.OriginalPrice = product.OriginalPrice
	response.Product.AmountCurrency = product.AmountCurrency
	response.Product.AmountUsd = product.AmountUsd
	response.Product.Images = imageURLs

	sdlog.Infof("获取订单详情V2成功，订单ID: %d", orderID)
	return webapi.OK(response).Render(ec)
}
