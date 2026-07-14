package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service/db"
	"TMA/service/db/model"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"github.com/sulink/ueapisdk/utils/constants"
	"gorm.io/gorm"
)

// parseOrderRequest 解析订单请求参数
// func parseOrderRequest(ec *middleware.AppRequestContext) *models.CreateOrderV2Req {
// 	payType, _ := strconv.Atoi(ec.Request().URL.Query().Get("paytype"))
// 	amount, _ := strconv.ParseFloat(ec.Request().URL.Query().Get("amount"), 64)
// 	unit, _ := strconv.Atoi(ec.Request().URL.Query().Get("unit"))

// 	return &models.CreateOrderV2Req{
// 		UEReqbaseV2:    models.NewUEReqbaseV2(),
// 		Paycode:        ec.Nu.Config.UE.PayCode,
// 		BusinessTypeid: ec.Nu.Config.UE.BusinessTypeid,
// 		Subject:        ec.Request().URL.Query().Get("title"),
// 		Body:           ec.Request().URL.Query().Get("title"),
// 		Orderno:        "order-" + utils.GetRandomString(10),
// 		NoticeUrl:      fmt.Sprintf("%s/api/v1/ue/notify", ec.Nu.Config.UE.MyDomain),
// 		OrderItems: []models.OrderItem{{
// 			Paytype: payType,
// 			Amount:  amount,
// 			Unit:    unit,
// 		}},
// 	}
// }

// createOrderV2 创建订单V2
// func createOrderV2(ec *middleware.AppRequestContext) error {
// 	req := parseOrderRequest(ec)
// 	ueConfig := models.UeConfigParam2{
// 		Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
// 		Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
// 		Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
// 	}

// 	resp, err := ue_api_v2.CreateOrderV2(*req, ueConfig)
// 	if err != nil {
// 		sdlog.Errorf("[CreateOrderV2] 创建订单失败: %v", err)
// 		return webapi.Error(common.ErrService).Render(ec)
// 	}

// 	if resp.Result <= 0 {
// 		sdlog.Errorf("[CreateOrderV2] 创建订单失败: %s", resp.Message)
// 		return webapi.Error(common.ErrService).Render(ec)
// 	}

// 	return webapi.OK(CreatePayOrderResponse{
// 		OrderID: req.Orderno,
// 		Paycode: req.Paycode,
// 	}).Render(ec)
// }

// notify 支付成功回调
func notify(ec *middleware.AppRequestContext) error {
	sdlog.Info("开始处理支付回调")
	body, _ := io.ReadAll(ec.Request().Body)
	accessKeyID := ec.Request().Header.Get("accesskeyid")
	signBody := ec.Request().Header.Get("signbody")
	aesIV := ec.Request().Header.Get("aesiv")

	// 使用正确的日志方法
	sdlog.Infof("支付回调参数: body=%s, accessKeyID=%s, signBody=%s, aesIV=%s", string(body), accessKeyID, signBody, aesIV)

	payHis, _ := ue_api_v2.GetResponse[models.PayHisV2Dto](
		string(body),
		signBody,
		ec.Nu.UeParam[constants.SERVER_PUBLICKEY].(string),
		ec.Nu.UeParam[constants.AES_KEY].(string),
		aesIV,
	)
	sdlog.Infof("支付回调数据: %+v", payHis)

	// 可能为产品订单，也可能是核销码订单
	var verificationCodeOrder model.VerificationCodeOrder
	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&verificationCodeOrder).Error
	if err != nil {
		sdlog.Errorf("查询核销码订单失败: %v", err)
	}
	if verificationCodeOrder.OrderID != 0 {
		sdlog.Infof("核销码订单存在，订单ID: %d", verificationCodeOrder.OrderID)
		un := &UENoticeBaseV2{}
		if un.ExecuteVerificationCodeOrder(ec, string(body), accessKeyID, signBody, aesIV, payHis) {
			sdlog.Info("支付回调处理成功")
			return ec.String(200, "OK")
		} else {
			sdlog.Error("支付回调处理失败")
			return ec.String(200, "Failed")
		}
	}

	var productOrder model.ProductOrder
	err = ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&productOrder).Error
	if err != nil {
		sdlog.Errorf("查询产品订单失败: %v", err)
	}
	if productOrder.OrderID != 0 {
		sdlog.Infof("产品订单存在，订单ID: %d", productOrder.OrderID)
		un := &UENoticeBaseV2{}
		if un.ExecuteProductOrder(ec, string(body), accessKeyID, signBody, aesIV, payHis) {
			sdlog.Info("支付回调处理成功")
			return ec.String(200, "OK")
		} else {
			sdlog.Error("支付回调处理失败")
			return ec.String(200, "Failed")
		}
	}

	var memberOrder model.MemberOrder
	err = ec.Nu.DB.Model(&model.MemberOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&memberOrder).Error
	if err != nil {
		sdlog.Errorf("查询会员订单失败: %v", err)
	}
	if memberOrder.OrderID != 0 {
		sdlog.Infof("会员订单存在，订单ID: %d", memberOrder.OrderID)
	}
	if memberOrder.OrderID != 0 {
		sdlog.Infof("会员订单存在，订单ID: %d", memberOrder.OrderID)
		un := &UENoticeBaseV2{}
		if un.ExecuteMemberOrder(ec, string(body), accessKeyID, signBody, aesIV, payHis) {
			sdlog.Info("支付回调处理成功")
			return ec.String(200, "OK")
		}
	}
	// un := &UENoticeBaseV2{}
	// if un.Execute(ec, string(body), accessKeyID, signBody, aesIV) {
	// 	sdlog.Info("支付回调处理成功")
	// 	return ec.String(200, "OK")
	// } else {
	// 	sdlog.Error("支付回调处理失败")
	// 	return ec.String(200, "Failed")
	// }

	return ec.String(200, "OK")
}

type UENoticeBaseV2 struct {
}

func (u *UENoticeBaseV2) Execute(ec *middleware.AppRequestContext, body, accesskeyid, signbody, aesiv string) bool {
	if ec.Nu.Config.UE.AccessKeyId != accesskeyid {
		return false
	}

	payHis, _ := ue_api_v2.GetResponse[models.PayHisV2Dto](
		body,
		signbody,
		ec.Nu.UeParam[constants.SERVER_PUBLICKEY].(string),
		ec.Nu.UeParam[constants.AES_KEY].(string),
		aesiv,
	)
	sdlog.Infof("支付回调数据: %+v", payHis)

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("ue_notify:lock:%s", payHis.Orderno)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return false
	}

	var orderItems []models.OrderItem
	json.Unmarshal([]byte(payHis.Orderitems), &orderItems)

	ueConfig := models.UeConfigParam2{
		Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
		Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
		Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
	}
	// 可能为产品订单，也可能是核销码订单
	// 查询订单信息
	var order model.VerificationCodeOrder
	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在或已处理: %d", payHis.Orderno)
			return false
		}
		sdlog.Errorf("Execute查询订单信息失败: %v", err)
		return false
	}

	if order.Status != common.VerificationCodeOrderStatusUnpaid {
		sdlog.Infof("订单已处理，订单ID: %d", order.OrderID)
		return true
	}

	// 从 orderItems 中获取支付金额
	var totalAmount float64
	for _, item := range orderItems {
		if item.Paytype == 25 {
			totalAmount += item.Amount
		}
	}

	if order.Amount != totalAmount {
		sdlog.Errorf("订单金额不匹配: %f != %f", order.Amount, totalAmount)
		return false
	}
	if ec.Nu.Config.UE.PayCode != payHis.Paycode {
		sdlog.Errorf("订单支付码不匹配: %s != %s", ec.Nu.Config.UE.PayCode, payHis.Paycode)
		return false
	}

	// 开启事务
	tx := ec.Nu.DB.Begin()
	if tx.Error != nil {
		sdlog.Errorf("开启事务失败: %v", tx.Error)
		return false
	}
	// 更新用户身份标签
	var product model.Product
	err = tx.Where("id = ?", order.ProductID).First(&product).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询产品信息失败: %v", err)
		return false
	}

	var user model.User
	err = tx.Where("user_id = ?", order.UserID).First(&user).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询用户信息失败: %v", err)
		return false
	}
	// 如果产品等级大于用户当前等级,更新用户身份标签
	if product.Level > user.IdentityTag {
		err = tx.Model(&model.User{}).
			Where("user_id = ?", order.UserID).
			Updates(map[string]interface{}{
				"identity_tag":  product.Level,
				"identity_name": product.LevelTag,
			}).Error
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("更新用户身份标签失败: %v", err)
			return false
		}
	}

	// 更新订单状态
	err = tx.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", order.OrderID).
		Updates(map[string]interface{}{
			"status":      common.VerificationCodeOrderStatusPaid,
			"ue_order_id": payHis.Ordernoue,
		}).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("更新订单状态失败: %v", err)
		return false
	}

	// 批量生成核销码
	var codes []model.VerificationCode
	for i := 0; i < int(order.CodeNum); i++ {
		code := model.VerificationCode{
			BuyUID:    order.UserID,
			OrderID:   order.OrderID,
			Code:      sdrand.GenerateSecureString(13, true, false, true),
			Status:    common.VerificationCodeStatusUnused,
			UseUID:    0,
			UseTime:   time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		codes = append(codes, code)
	}

	// 批量插入核销码
	if err := tx.Create(&codes).Error; err != nil {
		tx.Rollback()
		sdlog.Errorf("批量生成核销码失败: %v", err)
		return false
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return false
	}

	validateResult, _ := ue_api_v2.PayCallValidateV2(
		models.PayValidateReqV2{
			UEReqbaseV2: models.NewUEReqbaseV2(),
			Orderno:     strconv.FormatInt(order.OrderID, 10),
			Ordernoue:   payHis.Ordernoue,
			Paycode:     ec.Nu.Config.UE.PayCode,
			Orderitems:  orderItems,
		},
		ueConfig,
	)

	if validateResult.Result <= 0 {
		sdlog.Infof("支付回调校验失败, orderue=%s", payHis.Ordernoue)
		return false
	}

	sdlog.Infof("处理成功,订单号:%s,订单状态:%s", payHis.Orderno, payHis.Paystatus)
	return true
}

// openSecret 处理UE免密支付绑定回调
func openSecret(ec *middleware.AppRequestContext) error {
	sdlog.Info("收到UE免密绑定回调")
	body, _ := io.ReadAll(ec.Request().Body)
	accessKeyID := ec.Request().Header.Get("accesskeyid")
	signBody := ec.Request().Header.Get("signbody")
	aesIV := ec.Request().Header.Get("aesiv")

	sdlog.Infof("免密绑定回调参数: body=%s, accessKeyID=%s, signBody=%s, aesIV=%s", string(body), accessKeyID, signBody, aesIV)

	// 校验 accesskeyid
	if ec.Nu.Config.UE.AccessKeyId != accessKeyID {
		sdlog.Errorf("accesskeyid不匹配: %s != %s", accessKeyID, ec.Nu.Config.UE.AccessKeyId)
		return ec.String(200, "Failed")
	}

	// 解密验签
	secretDto, err := ue_api_v2.GetResponse[models.SecretDto](
		string(body),
		signBody,
		ec.Nu.UeParam[constants.SERVER_PUBLICKEY].(string),
		ec.Nu.UeParam[constants.AES_KEY].(string),
		aesIV,
	)
	if err != nil {
		sdlog.Errorf("免密绑定回调解密失败: %v", err)
		return ec.String(200, "Failed")
	}
	sdlog.Infof("免密绑定回调数据: opennodecode=%s, uenodecode=%s", secretDto.Opennodecode, secretDto.Uenodecode)

	// opennodecode 即 user_id
	userID, err := strconv.ParseInt(secretDto.Opennodecode, 10, 64)
	if err != nil {
		sdlog.Errorf("免密绑定回调 user_id 解析失败: opennodecode=%s", secretDto.Opennodecode)
		return ec.String(200, "Failed")
	}

	// 存储绑定关系
	err = db.BindUserUeSecret(ec.Nu.DB, userID, secretDto.Opennodecode, secretDto.Uenodecode)
	if err != nil {
		sdlog.Errorf("免密绑定存储失败: userID=%d, err=%v", userID, err)
		return ec.String(200, "Failed")
	}

	sdlog.Infof("免密绑定成功: userID=%d, uenodecode=%s", userID, secretDto.Uenodecode)
	return ec.String(200, "OK")
}

// closeSecret 处理UE免密支付解绑回调
func closeSecret(ec *middleware.AppRequestContext) error {
	sdlog.Info("收到UE免密解绑回调")
	body, _ := io.ReadAll(ec.Request().Body)
	accessKeyID := ec.Request().Header.Get("accesskeyid")
	signBody := ec.Request().Header.Get("signbody")
	aesIV := ec.Request().Header.Get("aesiv")

	sdlog.Infof("免密解绑回调参数: body=%s, accessKeyID=%s, signBody=%s, aesIV=%s", string(body), accessKeyID, signBody, aesIV)

	if ec.Nu.Config.UE.AccessKeyId != accessKeyID {
		sdlog.Errorf("accesskeyid不匹配: %s != %s", accessKeyID, ec.Nu.Config.UE.AccessKeyId)
		return ec.String(200, "Failed")
	}

	secretDto, err := ue_api_v2.GetResponse[models.SecretDto](
		string(body),
		signBody,
		ec.Nu.UeParam[constants.SERVER_PUBLICKEY].(string),
		ec.Nu.UeParam[constants.AES_KEY].(string),
		aesIV,
	)
	if err != nil {
		sdlog.Errorf("免密解绑回调解密失败: %v", err)
		return ec.String(200, "Failed")
	}
	sdlog.Infof("免密解绑回调数据: opennodecode=%s, uenodecode=%s", secretDto.Opennodecode, secretDto.Uenodecode)

	userID, err := strconv.ParseInt(secretDto.Opennodecode, 10, 64)
	if err != nil {
		sdlog.Errorf("免密解绑回调 user_id 解析失败: opennodecode=%s", secretDto.Opennodecode)
		return ec.String(200, "Failed")
	}

	err = db.UnbindUserUeSecret(ec.Nu.DB, userID)
	if err != nil {
		sdlog.Errorf("免密解绑失败: userID=%d, err=%v", userID, err)
		return ec.String(200, "Failed")
	}

	sdlog.Infof("免密解绑成功: userID=%d", userID)
	return ec.String(200, "OK")
}

// getSecretStatus 查询用户UE免密支付绑定状态
func getSecretStatus(ec *middleware.AppRequestContext) error {
	userIDStr := ec.QueryParams().Get("user_id")
	if userIDStr == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		sdlog.Errorf("查询免密绑定状态 user_id 解析失败: %s", userIDStr)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	_, err = db.GetUserUeSecret(ec.Nu.DB, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(map[string]interface{}{
				"is_bound": false,
			}).Render(ec)
		}
		sdlog.Errorf("查询免密绑定状态失败: userID=%d, err=%v", userID, err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(map[string]interface{}{
		"is_bound": true,
	}).Render(ec)
}

// 创建订单响应结构体
type CreatePayOrderResponse struct {
	OrderID string `json:"order_id"` // 订单ID
	Paycode string `json:"paycode"`  // 支付码
}

func (u *UENoticeBaseV2) ExecuteVerificationCodeOrder(ec *middleware.AppRequestContext, body, accesskeyid, signbody, aesiv string, payHis *models.PayHisV2Dto) bool {
	if ec.Nu.Config.UE.AccessKeyId != accesskeyid {
		return false
	}

	// payHis, _ := ue_api_v2.GetResponse[models.PayHisV2Dto](
	// 	body,
	// 	signbody,
	// 	ec.Nu.UeParam[constants.SERVER_PUBLICKEY].(string),
	// 	ec.Nu.UeParam[constants.AES_KEY].(string),
	// 	aesiv,
	// )
	// sdlog.Infof("支付回调数据: %+v", payHis)

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("ue_notify:lock:%s", payHis.Orderno)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return false
	}

	var orderItems []models.OrderItem
	json.Unmarshal([]byte(payHis.Orderitems), &orderItems)

	ueConfig := models.UeConfigParam2{
		Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
		Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
		Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
	}
	// 可能为产品订单，也可能是核销码订单
	// 查询订单信息
	var order model.VerificationCodeOrder
	err := ec.Nu.DB.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在或已处理: %d", payHis.Orderno)
			return false
		}
		sdlog.Errorf("Execute查询订单信息失败: %v", err)
		return false
	}

	if order.Status != common.VerificationCodeOrderStatusUnpaid {
		sdlog.Infof("订单已处理，订单ID: %d", order.OrderID)
		return true
	}

	// 从 orderItems 中获取支付金额
	var totalAmount float64
	for _, item := range orderItems {
		if item.Paytype == 25 {
			totalAmount += item.Amount
		}
	}

	if order.Amount != totalAmount {
		sdlog.Errorf("订单金额不匹配: %f != %f", order.Amount, totalAmount)
		return false
	}
	if ec.Nu.Config.UE.PayCode != payHis.Paycode {
		sdlog.Errorf("订单支付码不匹配: %s != %s", ec.Nu.Config.UE.PayCode, payHis.Paycode)
		return false
	}

	// 开启事务
	tx := ec.Nu.DB.Begin()
	if tx.Error != nil {
		sdlog.Errorf("开启事务失败: %v", tx.Error)
		return false
	}
	// 更新用户身份标签
	var product model.Product
	err = tx.Where("id = ?", order.ProductID).First(&product).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询产品信息失败: %v", err)
		return false
	}

	var user model.User
	err = tx.Where("user_id = ?", order.UserID).First(&user).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询用户信息失败: %v", err)
		return false
	}
	// 如果产品等级大于用户当前等级,更新用户身份标签
	if product.Level > user.IdentityTag {
		err = tx.Model(&model.User{}).
			Where("user_id = ?", order.UserID).
			Updates(map[string]interface{}{
				"identity_tag":  product.Level,
				"identity_name": product.LevelTag,
			}).Error
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("更新用户身份标签失败: %v", err)
			return false
		}
	}

	// 更新订单状态
	err = tx.Model(&model.VerificationCodeOrder{}).
		Where("order_id = ?", order.OrderID).
		Updates(map[string]interface{}{
			"status":      common.VerificationCodeOrderStatusPaid,
			"ue_order_id": payHis.Ordernoue,
		}).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("更新订单状态失败: %v", err)
		return false
	}

	// 批量生成核销码
	var codes []model.VerificationCode
	for i := 0; i < int(order.CodeNum); i++ {
		code := model.VerificationCode{
			BuyUID:    order.UserID,
			OrderID:   order.OrderID,
			Code:      sdrand.GenerateSecureString(13, true, false, true),
			Status:    common.VerificationCodeStatusUnused,
			UseUID:    0,
			UseTime:   time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		codes = append(codes, code)
	}

	// 批量插入核销码
	if err := tx.Create(&codes).Error; err != nil {
		tx.Rollback()
		sdlog.Errorf("批量生成核销码失败: %v", err)
		return false
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return false
	}

	validateResult, _ := ue_api_v2.PayCallValidateV2(
		models.PayValidateReqV2{
			UEReqbaseV2: models.NewUEReqbaseV2(),
			Orderno:     strconv.FormatInt(order.OrderID, 10),
			Ordernoue:   payHis.Ordernoue,
			Paycode:     ec.Nu.Config.UE.PayCode,
			Orderitems:  orderItems,
		},
		ueConfig,
	)

	if validateResult.Result <= 0 {
		sdlog.Infof("支付回调校验失败, orderue=%s", payHis.Ordernoue)
		return false
	}

	sdlog.Infof("处理成功,订单号:%s,订单状态:%s", payHis.Orderno, payHis.Paystatus)
	return true
}

func (u *UENoticeBaseV2) ExecuteProductOrder(ec *middleware.AppRequestContext, body, accesskeyid, signbody, aesiv string, payHis *models.PayHisV2Dto) bool {
	if ec.Nu.Config.UE.AccessKeyId != accesskeyid {
		return false
	}

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("ue_notify:lock:%s", payHis.Orderno)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return false
	}

	var orderItems []models.OrderItem
	json.Unmarshal([]byte(payHis.Orderitems), &orderItems)

	ueConfig := models.UeConfigParam2{
		Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
		Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
		Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
	}
	// 查询订单信息
	var order model.ProductOrder
	err := ec.Nu.DB.Model(&model.ProductOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在或已处理: %d", payHis.Orderno)
			return false
		}
		sdlog.Errorf("Execute查询订单信息失败: %v", err)
		return false
	}

	if order.Status != common.ProductOrderStatusPendingPayment && order.Status != common.ProductOrderStatusPaymentProcessing {
		sdlog.Infof("订单已处理，订单ID: %d", order.OrderID)
		return true
	}

	// 从 orderItems 中获取支付金额
	var totalAmount float64
	for _, item := range orderItems {
		if item.Paytype == 25 {
			totalAmount += item.Amount
		}
	}

	if order.Amount != totalAmount {
		sdlog.Errorf("订单金额不匹配: %f != %f", order.Amount, totalAmount)
		return false
	}
	if ec.Nu.Config.UE.PayCode != payHis.Paycode {
		sdlog.Errorf("订单支付码不匹配: %s != %s", ec.Nu.Config.UE.PayCode, payHis.Paycode)
		return false
	}

	// 开启事务
	tx := ec.Nu.DB.Begin()
	if tx.Error != nil {
		sdlog.Errorf("开启事务失败: %v", tx.Error)
		return false
	}

	var user model.User
	err = tx.Where("user_id = ?", order.UserID).First(&user).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询用户信息失败: %v", err)
		return false
	}

	// 更新订单状态
	err = tx.Model(&model.ProductOrder{}).
		Where("order_id = ?", order.OrderID).
		Updates(map[string]interface{}{
			"status":      common.ProductOrderStatusPendingShipment,
			"ue_order_id": payHis.Ordernoue,
		}).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("更新订单状态失败: %v", err)
		return false
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return false
	}

	validateResult, _ := ue_api_v2.PayCallValidateV2(
		models.PayValidateReqV2{
			UEReqbaseV2: models.NewUEReqbaseV2(),
			Orderno:     strconv.FormatInt(order.OrderID, 10),
			Ordernoue:   payHis.Ordernoue,
			Paycode:     ec.Nu.Config.UE.PayCode,
			Orderitems:  orderItems,
		},
		ueConfig,
	)

	if validateResult.Result <= 0 {
		sdlog.Infof("支付回调校验失败, orderue=%s", payHis.Ordernoue)
		return false
	}

	sdlog.Infof("处理成功,订单号:%s,订单状态:%s", payHis.Orderno, payHis.Paystatus)
	return true
}

func (u *UENoticeBaseV2) ExecuteMemberOrder(ec *middleware.AppRequestContext, body, accesskeyid, signbody, aesiv string, payHis *models.PayHisV2Dto) bool {
	if ec.Nu.Config.UE.AccessKeyId != accesskeyid {
		return false
	}

	// 获取锁防止频繁发送
	lockKey := fmt.Sprintf("ue_notify:lock:%s", payHis.Orderno)
	if ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockKey); !ok {
		return false
	}

	var orderItems []models.OrderItem
	json.Unmarshal([]byte(payHis.Orderitems), &orderItems)

	ueConfig := models.UeConfigParam2{
		Aeskey:      ec.Nu.UeParam[constants.AES_KEY].(string),
		Privatekey:  ec.Nu.UeParam[constants.RSA_PRIVATEKEY].(string),
		Accesstoken: ec.Nu.UeParam[constants.ACCESS_TOKEN].(string),
	}
	// 查询订单信息
	var order model.MemberOrder
	err := ec.Nu.DB.Model(&model.MemberOrder{}).
		Where("order_id = ?", payHis.Orderno).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sdlog.Errorf("订单不存在或已处理: %d", payHis.Orderno)
			return false
		}
		sdlog.Errorf("Execute查询订单信息失败: %v", err)
		return false
	}

	if order.Status != common.MemberOrderStatusUnpaid {
		sdlog.Infof("订单已处理，订单ID: %d", order.OrderID)
		return true
	}

	// 从 orderItems 中获取支付金额
	var totalAmount float64
	for _, item := range orderItems {
		if item.Paytype == 25 {
			totalAmount += item.Amount
		}
	}

	if order.Amount != totalAmount {
		sdlog.Errorf("订单金额不匹配: %f != %f", order.Amount, totalAmount)
		return false
	}
	if ec.Nu.Config.UE.PayCode != payHis.Paycode {
		sdlog.Errorf("订单支付码不匹配: %s != %s", ec.Nu.Config.UE.PayCode, payHis.Paycode)
		return false
	}

	// 开启事务
	tx := ec.Nu.DB.Begin()
	if tx.Error != nil {
		sdlog.Errorf("开启事务失败: %v", tx.Error)
		return false
	}

	var user model.User
	err = tx.Where("user_id = ?", order.UserID).First(&user).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("查询用户信息失败: %v", err)
		return false
	}

	// 更新订单状态
	err = tx.Model(&model.MemberOrder{}).
		Where("order_id = ?", order.OrderID).
		Updates(map[string]interface{}{
			"status":      common.MemberOrderStatusPaid,
			"ue_order_id": payHis.Ordernoue,
		}).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("更新订单状态失败: %v", err)
		return false
	}

	// 更新用户会员到期时间， 根据订单信息，传入订单ID
	err = db.UpdateUserMemberByMemberOrder(tx, order.OrderID)
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("更新用户会员到期时间失败: %v", err)
		return false
	}
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		sdlog.Errorf("提交事务失败: %v", err)
		return false
	}

	validateResult, _ := ue_api_v2.PayCallValidateV2(
		models.PayValidateReqV2{
			UEReqbaseV2: models.NewUEReqbaseV2(),
			Orderno:     strconv.FormatInt(order.OrderID, 10),
			Ordernoue:   payHis.Ordernoue,
			Paycode:     ec.Nu.Config.UE.PayCode,
			Orderitems:  orderItems,
		},
		ueConfig,
	)

	if validateResult.Result <= 0 {
		sdlog.Infof("支付回调校验失败, orderue=%s", payHis.Ordernoue)
		return false
	}

	sdlog.Infof("处理成功,订单号:%s,订单状态:%s", payHis.Orderno, payHis.Paystatus)
	return true
}
