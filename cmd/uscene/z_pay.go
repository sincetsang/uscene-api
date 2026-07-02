package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
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

type ResponseAddressCheckpoint struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Chain            string `gorm:"column:chain;not null" json:"chain"`
	Currency         string `gorm:"column:currency;not null" json:"currency"`
	ContractAddress  string `gorm:"column:contract_address;not null" json:"contract_address"`
	ReceivingAddress string `gorm:"column:address;not null" json:"receiving_address"`
	TokenDecimals    int32  `gorm:"column:token_decimals;not null" json:"token_decimals"`
	Description      string `gorm:"column:description;not null" json:"description"`
}

// 查询支付方式
func getPayMethodList(ec *middleware.AppRequestContext) error {
	var list []model.AddressCheckpoint
	db := ec.Nu.DB
	err := db.Where("is_active = ?", true).Find(&list).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrParam).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 转换为响应格式
	var responseList []ResponseAddressCheckpoint
	for _, item := range list {
		responseList = append(responseList, ResponseAddressCheckpoint{
			ID:               item.ID,
			Chain:            item.Chain,
			Currency:         item.Currency,
			ReceivingAddress: item.Address,
			ContractAddress:  item.ContractAddress,
			TokenDecimals:    item.TokenDecimals,
			Description:      item.Description,
		})
	}

	return webapi.OK(responseList).Render(ec)
}

// PayOrderResponseV2 支付订单响应
type PayOrderResponseV2 struct {
	OrderID          string  `json:"order_id"` // 订单ID
	Paycode          string  `json:"paycode"`  // 支付码
	PayType          string  `json:"pay_type"`
	Currency         string  `json:"currency"`
	ProductNum       int32   `json:"product_num"`
	ProductPrice     float64 `json:"product_price"`
	Amount           float64 `json:"amount"`
	Value            string  `json:"value"`
	ReceivingAddress string  `json:"receiving_address"`
	ContractAddress  string  `json:"contract_address"`
}

// getPayInfoV2 获取支付信息
func getPayInfoV2(ec *middleware.AppRequestContext) error {
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
	if order.PayType == "FEC" {
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
			ProductNum:   order.ProductNum,
			ProductPrice: decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
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
					BizType:     common.PaymentBizTypeCode,
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

		sdlog.Infof("获取支付信息成功，订单ID: %s", orderID)
		return webapi.OK(PayOrderResponseV2{
			OrderID:          payOrder.OrderID,
			Paycode:          "",
			Amount:           decimal.NewFromFloat(payOrder.Amount).InexactFloat64(),
			ProductNum:       order.ProductNum,
			ProductPrice:     decimal.NewFromFloat(order.ProductPrice).InexactFloat64(),
			PayType:          payOrder.Chain,
			Currency:         payOrder.Currency,
			Value:            ConvertAmountToValue(payOrder.Amount, int(address.TokenDecimals)),
			ReceivingAddress: payOrder.ToAddress,
			ContractAddress:  address.ContractAddress,
		}).Render(ec)
	}
}

// ConvertAmountToValue 用于将订单金额（float64，实际金额）根据 token 的小数位数转换为区块链金额（字符串表示）。
func ConvertAmountToValue(amount float64, tokenDecimals int) string {
	// 使用 decimal.NewFromFloat 创建 decimal
	amt := decimal.NewFromFloat(amount)

	// 先舍入到 tokenDecimals + 2 位小数（额外精度用于恢复精度），然后再舍入到 tokenDecimals 位
	// 这样可以确保像 4.18 这样的值能够正确恢复精度
	// 例如：4.18 在 float64 中可能是 4.1799999999999997，但经过两次舍入后可以恢复为 4.18
	amt = amt.Round(int32(tokenDecimals + 2)).Round(int32(tokenDecimals))

	// 构建 multiplier = 10^decimals
	multiplier := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(tokenDecimals)))

	// 乘以倍数（精确计算最小单位）
	result := amt.Mul(multiplier)

	// 转换为 big.Int，向下取整（链上不允许小数）
	return result.BigInt().String()
}
