package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// 账单响应结构体
type BillResponse struct {
	TotalCount int64                     `json:"total_count"` // 总数
	Bills      []UserAssetDetailResponse `json:"bills"`       // 账单列表
}

type UserAssetDetailResponse struct {
	Type      string          `json:"type"`       // 类型
	Currency  string          `json:"currency"`   // 货币
	Amount    decimal.Decimal `json:"amount"`     // 金额
	AmountStr string          `json:"amount_str"` // 金额字符串
	CreatedAt time.Time       `json:"created_at"` // 创建时间
}

// 提现请求参数
type WithdrawRequest struct {
	Amount   decimal.Decimal `json:"amount"`   // 提现金额
	Currency string          `json:"currency"` // 提现货币
	Address  string          `json:"address"`  // 提现地址
}

// 提现响应
type WithdrawResponse struct {
	OrderID               string    `json:"order_id"`                // 提现订单ID
	Value                 string    `json:"value"`                   // 提现值
	Currency              string    `json:"currency"`                // 货币
	WithdrawStatus        string    `json:"withdraw_status"`         // 提现状态
	WithdrawWalletAddress string    `json:"withdraw_wallet_address"` // 提现钱包地址
	WithdrawAt            time.Time `json:"withdraw_at"`             // 提现时间
	CreatedAt             time.Time `json:"created_at"`              // 创建时间
	UpdatedAt             time.Time `json:"updated_at"`              // 更新时间
}

// 获取用户账单
func userBill(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 获取分页参数
	page, _ := strconv.Atoi(ec.QueryParams().Get("page"))
	pageSize, _ := strconv.Atoi(ec.QueryParams().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建查询条件
	query := ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Where("user_id = ?", userId)

	// 获取总数
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var bills []model.UserAssetDetail
	err = query.Where("user_id = ?", userId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&bills).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建响应
	var response []UserAssetDetailResponse
	language := common.GetRequestLanguage()
	for _, bill := range bills {
		amount := decimal.NewFromFloat(bill.Rewards)
		switch bill.Category {
		case common.UserAssetCategoryAdvertisement:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "广告奖励",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Advertisement reward",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		case common.UserAssetCategoryWithdraw:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "提现",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Withdraw",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		}
	}

	return webapi.OK(BillResponse{
		TotalCount: total,
		Bills:      response,
	}).Render(ec)
}

// 获取用户账单
func userBillV2(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 获取分页参数
	page, _ := strconv.Atoi(ec.QueryParams().Get("page"))
	pageSize, _ := strconv.Atoi(ec.QueryParams().Get("page_size"))
	currency := ec.QueryParams().Get("currency")
	if currency == "" {
		currency = "FEC"
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建查询条件
	query := ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Where("user_id = ?", userId).
		Where("currency = ?", currency)

	// 获取总数
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var bills []model.UserAssetDetail
	err = query.Where("user_id = ?", userId).Where("currency = ?", currency).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&bills).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建响应
	var response []UserAssetDetailResponse
	language := common.GetRequestLanguage()
	for _, bill := range bills {
		amount := decimal.NewFromFloat(bill.Rewards)
		switch bill.Category {
		case common.UserAssetCategoryAdvertisement:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "广告奖励",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Advertisement reward",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		case common.UserAssetCategoryWithdraw:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "提现",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Withdraw",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		case common.UserAssetCategoryRefund:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "退款",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Refund",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		case common.UserAssetCategorySold:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "出售商品",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Sold product",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "+" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		case common.UserAssetCategoryRefundToUser:
			if language == common.LanguageZh {
				response = append(response, UserAssetDetailResponse{
					Type:      "退款给用户",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			} else {
				response = append(response, UserAssetDetailResponse{
					Type:      "Refund to user",
					Currency:  bill.Currency,
					Amount:    amount,
					AmountStr: "-" + amount.String() + " " + bill.Currency,
					CreatedAt: bill.CreatedAt,
				})
			}
		}
	}

	return webapi.OK(BillResponse{
		TotalCount: total,
		Bills:      response,
	}).Render(ec)
}

// 用户提现
func userWithdraw(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 解析请求参数
	req := WithdrawRequest{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查提现金额
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 使用Redis加锁
	lockKey := fmt.Sprintf("user_balance:lock:%d", userId)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}

	// 检查用户是否已有进行中的提现订单
	var existingOrder model.UserWithdrawOrder
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ? AND withdraw_status = ?", userId, common.WithdrawStatusPending).
		First(&existingOrder).Error
	if err == nil {
		return webapi.Error(common.ErrWithdrawInProgress).Render(ec)
	}

	// 开始数据库事务
	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	if tx.Error != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查用户余额
	var userAsset model.UserAsset
	err = tx.Where("user_id = ?", userId).
		Where("currency = ?", req.Currency).
		First(&userAsset).Error
	if err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	userBalance := decimal.NewFromFloat(userAsset.Balance)
	if userBalance.LessThan(req.Amount) {
		tx.Rollback()
		return webapi.Error(common.ErrInsufficientBalance).Render(ec)
	}

	// 创建提现订单
	order := model.UserWithdrawOrder{
		OrderID:               sdrand.GenerateSecureString(13, true, false, true),
		UserID:                userId,
		Amount:                int64(0),
		Value:                 req.Amount.String(),
		Currency:              req.Currency,
		WithdrawStatus:        common.WithdrawStatusPending,
		WithdrawWalletAddress: req.Address,
		WithdrawAt:            time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = tx.Create(&order).Error
	if err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新用户余额
	newBalance := userBalance.Sub(req.Amount)
	err = tx.Model(&model.UserAsset{}).
		Where("user_id = ?", userId).
		Where("currency = ?", req.Currency).
		Update("balance", newBalance.InexactFloat64()).Error
	if err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 记录提现明细
	record := model.UserAssetDetail{
		UserID:    userId,
		RelatedID: order.ID,
		Category:  common.UserAssetCategoryWithdraw,
		Rewards:   req.Amount.InexactFloat64(),
		Currency:  req.Currency,
	}
	err = tx.Create(&record).Error
	if err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(WithdrawResponse{
		OrderID:               order.OrderID,
		Value:                 order.Value,
		Currency:              order.Currency,
		WithdrawStatus:        order.WithdrawStatus,
		WithdrawWalletAddress: order.WithdrawWalletAddress,
		WithdrawAt:            order.WithdrawAt,
		CreatedAt:             order.CreatedAt,
		UpdatedAt:             order.UpdatedAt,
	}).Render(ec)
}

// 查询最新的一笔进行中的提现订单信息
func userLatestWithdrawOrder(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 查询最新的一笔进行中的提现订单
	var latestOrder model.UserWithdrawOrder
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ? AND withdraw_status = ?", userId, common.WithdrawStatusPending).
		Order("created_at DESC").
		First(&latestOrder).Error
	if err != nil {
		// 如果未查询到订单，返回空响应
		return webapi.OK(WithdrawResponse{}).Render(ec)
	}

	// 构建响应
	response := WithdrawResponse{
		OrderID:               latestOrder.OrderID,
		Value:                 latestOrder.Value,
		Currency:              latestOrder.Currency,
		WithdrawStatus:        latestOrder.WithdrawStatus,
		WithdrawWalletAddress: latestOrder.WithdrawWalletAddress,
		WithdrawAt:            latestOrder.WithdrawAt,
		CreatedAt:             latestOrder.CreatedAt,
		UpdatedAt:             latestOrder.UpdatedAt,
	}

	return webapi.OK(response).Render(ec)
}
