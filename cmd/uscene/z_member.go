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
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"github.com/sulink/ueapisdk/utils/constants"
	"gorm.io/gorm"
)

type MemberConfig struct {
	ID              int64   `json:"id"`
	Period          string  `json:"period"`
	Description     string  `json:"description"`
	SortIndex       int64   `json:"sort_index"`
	FecPrice        float64 `json:"fec_price"`
	FecOriginPrice  float64 `json:"fec_origin_price"`
	UsdtPrice       float64 `json:"usdt_price"`
	UsdtOriginPrice float64 `json:"usdt_origin_price"`
}

// GET /api/v1/member/config  table: supermap_checkinmemberconfig
func getMemberConfig(ec *middleware.AppRequestContext) error {
	var memberConfig []model.SupermapCheckinmemberconfig
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapCheckinmemberconfig{}).Order("sort_index ASC").Find(&memberConfig).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var memberConfigList []MemberConfig
	for _, config := range memberConfig {
		memberConfigList = append(memberConfigList, MemberConfig{
			ID:              config.ID,
			Period:          config.Period,
			Description:     config.Description,
			SortIndex:       int64(config.SortIndex),
			FecPrice:        config.FecPrice,
			FecOriginPrice:  config.FecOriginPrice,
			UsdtPrice:       config.UsdtPrice,
			UsdtOriginPrice: config.UsdtOriginPrice,
		})
	}
	return webapi.OK(memberConfigList).Render(ec)
}

// POST /api/v1/member/order/create  table: supermap_checkinmemberorder
// 逻辑与product order 相同，只是表名不同
// 检查之前是否有待支付订单，如果有则返回错误
type CreateMemberOrderRequest struct {
	MemberID int64  `json:"member_id"` // 会员ID
	PayType  string `json:"pay_type"`  // 支付方式 FEC,ETH
}

type CreateMemberOrderResponse struct {
	OrderID     int64     `json:"order_id"`     // 订单ID
	MemberID    int64     `json:"member_id"`    // 会员ID
	MemberPrice float64   `json:"member_price"` // 会员价格
	Amount      float64   `json:"amount"`       // 订单金额
	PayType     string    `json:"pay_type"`     // 支付方式
	Currency    string    `json:"currency"`     // 货币
	CreatedAt   time.Time `json:"created_at"`
	ExpiredAt   time.Time `json:"expired_at"`  // 订单过期时间
	Period      string    `json:"period"`      // 会员周期
	Description string    `json:"description"` // 会员描述
	Status      string    `json:"status"`      // 订单状态
}

func createMemberOrder(ec *middleware.AppRequestContext) error {

	var request CreateMemberOrderRequest
	if err := ec.Bind(&request); err != nil {
		if err != nil {
			sdlog.Errorf("解析请求参数失败: %v", err)
			return webapi.Error(common.ErrParam).Render(ec)
		}
	}

	if request.MemberID == 0 || request.PayType == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if request.PayType != "FEC" && request.PayType != "ETH" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查之前是否有待支付订单，如果有则返回错误
	var lastOrder model.MemberOrder
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.MemberOrder{}).Where("user_id = ? AND pay_type = ? AND status = ?", ec.AuthData.User.ID, request.PayType, common.MemberOrderStatusUnpaid).First(&lastOrder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if lastOrder.ID > 0 {
		return webapi.Error(common.ErrMemberOrderAlreadyExists).Render(ec)
	}

	// 获取会员价格
	var memberConfig model.SupermapCheckinmemberconfig
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapCheckinmemberconfig{}).Where("id = ?", request.MemberID).First(&memberConfig).Error
	if err != nil {
		sdlog.Errorf("获取会员价格失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 创建订单 参考createproductorder
	memberPrice := memberConfig.FecPrice
	unit := "FEC"
	if strings.ToUpper(request.PayType) == "ETH" {
		memberPrice = memberConfig.UsdtPrice
		unit = "USDT"
	}
	orderID, _ := strconv.ParseInt(sdrand.GenerateSecureString(10, true, false, false), 10, 64)
	order := &model.MemberOrder{
		OrderID:        orderID,
		UserID:         ec.AuthData.User.ID,
		PayType:        request.PayType,
		Period:         memberConfig.Period,
		Status:         common.MemberOrderStatusUnpaid,
		Amount:         memberPrice,
		ProductPrice:   decimal.NewFromFloat(memberPrice).InexactFloat64(),
		Unit:           unit,
		MemberConfigID: int32(request.MemberID),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = ec.Nu.DB.Create(order).Error
	if err != nil {
		sdlog.Errorf("创建订单失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("创建订单成功，订单ID: %d", order.ID)
	return webapi.OK(CreateMemberOrderResponse{
		OrderID:     order.OrderID,
		MemberID:    request.MemberID,
		MemberPrice: memberPrice,
		Amount:      decimal.NewFromFloat(memberPrice).InexactFloat64(),
		PayType:     request.PayType,
		Currency:    unit,
		CreatedAt:   order.CreatedAt.Truncate(time.Second),
		ExpiredAt:   order.CreatedAt.Truncate(time.Second).Add(15 * time.Minute),
		Period:      memberConfig.Period,
		Description: memberConfig.Description,
		Status:      order.Status,
	}).Render(ec)
}

type MemberOrderDetailResponse struct {
	OrderID      int64     `json:"order_id"`      // 订单ID
	ProductPrice float64   `json:"product_price"` // 会员价格
	ProductNum   int32     `json:"product_num"`   // 产品数量
	Amount       float64   `json:"amount"`        // 订单金额
	PayType      string    `json:"pay_type"`      // 支付方式
	Currency     string    `json:"currency"`      // 货币
	CreatedAt    time.Time `json:"created_at"`
	ExpiredAt    time.Time `json:"expired_at"`  // 订单过期时间
	Status       string    `json:"status"`      // 订单状态
	Period       string    `json:"period"`      // 会员周期
	Description  string    `json:"description"` // 会员描述
}

func getMemberOrderDetail(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取会员订单详情")

	// 获取订单ID
	orderID := ec.QueryParams().Get("order_id")
	if orderID == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		sdlog.Error("订单ID格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.MemberOrder
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.MemberOrder{}).Where("order_id = ?", orderIDInt).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrMemberOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	//
	var memberConfig model.SupermapCheckinmemberconfig
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SupermapCheckinmemberconfig{}).Where("id = ?", order.MemberConfigID).First(&memberConfig).Error
	if err != nil {
		sdlog.Errorf("查询会员配置失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(MemberOrderDetailResponse{
		OrderID:      order.OrderID,
		ProductPrice: decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
		ProductNum:   1,
		Amount:       decimal.NewFromFloat(order.Amount).InexactFloat64(),
		PayType:      order.PayType,
		Currency:     order.Unit,
		CreatedAt:    order.CreatedAt,
		ExpiredAt:    order.CreatedAt.Add(15 * time.Minute),
		Status:       order.Status,
		Period:       order.Period,
		Description:  memberConfig.Description,
	}).Render(ec)

}

func memberPayReport(ec *middleware.AppRequestContext) error {
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

	order := model.MemberOrder{}
	err := ec.Nu.DB.Model(&model.MemberOrder{}).
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

	if order.Status != common.MemberOrderStatusUnpaid {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	order.Status = common.MemberOrderStatusPaymentProcessing
	err = ec.Nu.DB.Model(&order).Update("status", common.MemberOrderStatusPaymentProcessing).Error
	if err != nil {
		sdlog.Errorf("更新订单状态失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 支付上报成功", orderID)
	return webapi.OK(true).Render(ec)
}

func getMemberOrderPayStatus(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取会员订单支付状态")

	// 获取订单ID
	orderID := ec.QueryParams().Get("order_id")
	if orderID == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		sdlog.Error("订单ID格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.MemberOrder
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.MemberOrder{}).Where("order_id = ?", orderIDInt).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrMemberOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(GetOrderPayStatusResponse{
		OrderID: order.OrderID,
		Status:  order.Status,
	}).Render(ec)
}

type CancelMemberOrderRequest struct {
	OrderID int64 `json:"order_id"` // 订单ID
}

func cancelMemberOrder(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始取消会员订单")

	var request CancelMemberOrderRequest
	if err := ec.Bind(&request); err != nil {
		sdlog.Errorf("解析请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if request.OrderID == 0 {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.MemberOrder
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.MemberOrder{}).Where("order_id = ?", request.OrderID).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", request.OrderID)
			return webapi.Error(common.ErrMemberOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证订单状态
	if order.Status != common.MemberOrderStatusUnpaid {
		sdlog.Errorf("订单状态不正确: %d", order.Status)
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	// 验证订单所属
	if order.UserID != ec.AuthData.User.ID {
		sdlog.Error("无权访问此订单")
		return webapi.Error(common.ErrUnPower).Render(ec)
	}

	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	// 更新订单状态为已取消
	err = tx.Model(&order).Update("status", common.MemberOrderStatusCancel).Error
	if err != nil {
		sdlog.Errorf("取消订单失败: %v", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 更新支付订单状态为已取消
	err = tx.Model(&model.PaymentOrder{}).Where("order_id = ?", request.OrderID).Update("status", common.OrderStatusCancel).Error
	if err != nil {
		sdlog.Errorf("取消支付订单失败: %v", err)
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 提交事务
	err = tx.Commit().Error
	if err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("订单 %s 取消成功", request.OrderID)
	return webapi.OK(true).Render(ec)
}

func getMemberOrderPayInfo(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取支付信息")

	// 获取订单ID
	orderID := ec.QueryParams().Get("order_id")
	if orderID == "" {
		sdlog.Error("订单ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询订单信息
	var order model.MemberOrder
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.MemberOrder{}).Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在: %s", orderID)
			return webapi.Error(common.ErrMemberOrderNotExist).Render(ec)
		}
		sdlog.Errorf("查询订单信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 验证订单状态
	if order.Status != common.MemberOrderStatusUnpaid {
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
			Body:           "member",
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
			ProductNum:   1,
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
					BizType:     common.PaymentBizTypeMember,
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

		response := PayOrderResponseV2{
			OrderID:          payOrder.OrderID,
			Paycode:          "",
			Amount:           decimal.NewFromFloat(payOrder.Amount).InexactFloat64(),
			ProductNum:       1,
			ProductPrice:     decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
			PayType:          payOrder.Chain,
			Currency:         payOrder.Currency,
			Value:            ConvertAmountToValue(payOrder.Amount, int(address.TokenDecimals)),
			ReceivingAddress: payOrder.ToAddress,
			ContractAddress:  address.ContractAddress,
		}
		sdlog.Infof("获取支付信息成功，订单ID: %s", orderID)
		sdlog.Infof("获取支付信息成功，订单ID: %f, %s", response.Amount, response.Value)
		return webapi.OK(response).Render(ec)
	}
}

type UserMemberInfo struct {
	IsVip      bool  `json:"is_vip"`
	ExpireTime int32 `json:"expire_time"`
}

func getUserMemberInfo(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取用户会员信息")

	// 获取用户ID
	userID := ec.AuthData.User.ID
	if userID == 0 {
		sdlog.Error("用户ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询用户会员信息
	var member model.SupermapCheckinmember
	err := ec.Nu.DB.Model(&model.SupermapCheckinmember{}).Where("user_id = ?", userID).First(&member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(UserMemberInfo{
				IsVip:      false,
				ExpireTime: 0,
			}).Render(ec)
		}
		sdlog.Errorf("查询用户会员信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	sdlog.Infof("用户会员信息: %+v", member)
	// 比较过期时间与当前时间，如果过期时间小于当前时间，则用户不是会员
	if member.ExpiredTimestamp < int32(time.Now().Unix()) {
		return webapi.OK(UserMemberInfo{
			IsVip:      false,
			ExpireTime: member.ExpiredTimestamp,
		}).Render(ec)
	}
	result := UserMemberInfo{
		IsVip:      true,
		ExpireTime: member.ExpiredTimestamp,
	}
	return webapi.OK(result).Render(ec)
}

// 会员购买记录
type MemberPurchaseRecord struct {
	Unit      string `json:"unit"`
	Price     string `json:"price"`
	Amount    string `json:"amount"`
	Timestamp int32  `json:"timestamp"`
	Period    string `json:"period"`
	Status    string `json:"status"`
	OrderID   int64  `json:"order_id"`
	PayType   string `json:"pay_type"`
}

type MemberPurchaseRecordList struct {
	Total    int64                  `json:"total"`
	Page     int64                  `json:"page"`
	PageSize int64                  `json:"page_size"`
	Records  []MemberPurchaseRecord `json:"records"`
}

// GET /api/v1/user/member/order/list  table: member_order
// 获取用户会员购买记录
// 分页查询，每页10条，查询第几页
// 查询参数：page, page_size
func getUserMemberOrderList(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始获取用户会员购买记录")

	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}
	pageInt, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		sdlog.Error("页码格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}
	pageSizeInt, err := strconv.ParseInt(pageSize, 10, 64)
	if err != nil {
		sdlog.Error("每页数量格式错误")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId == 0 {
		sdlog.Error("用户ID不能为空")
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var total int64
	err = ec.Nu.DB.Model(&model.MemberOrder{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		sdlog.Errorf("查询用户会员购买记录总数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var memberOrders []model.MemberOrder
	err = ec.Nu.DB.Model(&model.MemberOrder{}).Where("user_id = ?", userId).Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Order("created_at DESC").Find(&memberOrders).Error
	if err != nil {
		sdlog.Errorf("查询用户会员购买记录失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var address model.AddressCheckpoint
	err = ec.Nu.DB.Model(&model.AddressCheckpoint{}).
		Where("currency = ?", "FEC").
		Where("chain = ?", "ETH").
		First(&address).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("查询地址失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var memberPurchaseRecords []MemberPurchaseRecord
	for _, order := range memberOrders {
		var price string = ""
		switch order.PayType {
		case "FEC":
			price = strconv.FormatFloat(order.Amount, 'f', -1, 64)
		case "ETH":
			if strings.ToUpper(order.Unit) == "USDT" {
				price = strconv.FormatFloat(order.Amount, 'f', -1, 64)
			} else {
				price = ConvertAmountToValue(order.Amount, int(address.TokenDecimals))
			}
		}
		memberPurchaseRecords = append(memberPurchaseRecords, MemberPurchaseRecord{
			Unit:      order.Unit,
			Price:     price,
			Amount:    price,
			Timestamp: int32(order.CreatedAt.Unix()),
			Period:    order.Period + " days",
			Status:    order.Status,
			OrderID:   order.OrderID,
			PayType:   order.PayType,
		})
	}
	return webapi.OK(MemberPurchaseRecordList{
		Total:    total,
		Page:     pageInt,
		PageSize: pageSizeInt,
		Records:  memberPurchaseRecords,
	}).Render(ec)
}
