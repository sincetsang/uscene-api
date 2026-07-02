package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/web/webapi"
	"TMA/service/db"
	"TMA/service/db/model"
	"strconv"
)

// premiumInfo 用户会员状态 支付信息 什么币种什么价格
type ResPremiumInfo struct {
	IsPremium       bool   `json:"is_premium"`        // 会员状态
	Currency        string `json:"currency"`          // 支付币种
	Amount          int64  `json:"amount"`            // 支付金额(最小单位)
	Value           string `json:"value"`             // 显示金额
	Address         string `json:"address"`           // 收款地址
	HashRateRewards string `json:"hash_rate_rewards"` // 奖励的算力
}

func premiumInfo(ec *middleware.AppRequestContext) error {
	// 获取支付配置信息
	configs, err := db.GetConfigs(ec.Nu.DB.WithContext(ec.Request().Context()))
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var address = ""
	var denom = ""
	var amount int64
	var priceReal = ""
	var rewardPercent = ""

	for _, config := range configs {
		if config.Key == common.ConfigPrimaryAcceptAddress {
			address = config.Value
		} else if config.Key == common.ConfigPrimaryDenom {
			denom = config.Value
		} else if config.Key == common.ConfigPrimaryPrice {
			amount, err = strconv.ParseInt(config.Value, 10, 64)
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else if config.Key == common.ConfigPrimaryPriceReal {
			priceReal = config.Value
		} else if config.Key == common.ConfigPrimaryHashRateRewards {
			rewardPercent = config.Value
		}
	}

	response := ResPremiumInfo{
		IsPremium:       ec.AuthData.User.IsPremium,
		Currency:        denom,
		Amount:          amount,
		Value:           priceReal,
		Address:         address,
		HashRateRewards: rewardPercent,
	}

	return webapi.OK(response).Render(ec)
}

type ResAddSpotOrderInfo struct {
	OrderId       int64  `json:"order_id"`
	Currency      string `json:"currency"`
	Amount        int64  `json:"amount"`
	Value         string `json:"value"`
	SOLPayAddress string `json:"sol_pay_address"`
	Comment       string `json:"comment"` // 订单备注
}

func createPremiumOrder(ec *middleware.AppRequestContext) error {

	configs, errC := db.GetConfigs(ec.Nu.DB.WithContext(ec.Request().Context()))
	if errC != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var address = ""
	var denom = ""
	var amount int64
	var priceReal = ""
	for _, config := range configs {
		if config.Key == common.ConfigPrimaryAcceptAddress {
			address = config.Value
		} else if config.Key == common.ConfigPrimaryDenom {
			denom = config.Value
		} else if config.Key == common.ConfigPrimaryPrice {
			var err error
			amount, err = strconv.ParseInt(config.Value, 10, 64)
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else if config.Key == common.ConfigPrimaryPriceReal {
			priceReal = config.Value
		}
	}

	userId := ec.AuthData.User.ID

	// 开启事务
	tx := ec.Nu.DB.WithContext(ec.Request().Context()).Begin()
	if tx.Error != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建订单
	createOrder := model.PremiumOrder{
		UserID:        userId,
		OrderID:       0,
		Comment:       "",
		OrderFinished: false,
		Amount:        amount,
		Currency:      denom,
		Value:         priceReal,
		SolAddress:    address,
	}

	// 在事务中创建
	if err := tx.Create(&createOrder).Error; err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新 order_id 为自增 ID
	if err := tx.Model(&createOrder).Update("order_id", createOrder.ID).Error; err != nil {
		tx.Rollback()
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 将数据转换为响应格式
	response := ResAddSpotOrderInfo{
		OrderId:       createOrder.OrderID,
		Currency:      createOrder.Currency,
		Amount:        createOrder.Amount,
		Value:         createOrder.Value,
		SOLPayAddress: createOrder.SolAddress,
		Comment:       createOrder.Comment,
	}

	return webapi.OK(response).Render(ec)
}
