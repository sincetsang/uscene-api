package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/service/db/model"
	"context"
	"fmt"
	"time"
)

const (
	verificationOrderUpdateLockKey = "verification_order_update_lock"
	verificationOrderUpdateLockTTL = 10 * time.Minute
)

// 定义关联查询结果的结构体
type PaymentOrderWithVerificationOrder struct {
	model.PaymentOrder
	CodeNum   int32 `gorm:"column:code_num"`
	ProductID int64 `gorm:"column:product_id"`
	UserID    int64 `gorm:"column:user_id"`
	VOrderID  int64 `gorm:"column:v_order_id"` // verification_code_order的order_id
}

func verificationOrderUpdate(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, verificationOrderUpdateLockKey, lockValue, verificationOrderUpdateLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, verificationOrderUpdateLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	// 查询所有已支付但未回调的订单，关联verification_code_order表
	var paymentOrders []PaymentOrderWithVerificationOrder
	err = nu.DB.WithContext(ctx).
		Table("payment_orders").
		Select("payment_orders.*, verification_code_order.code_num, verification_code_order.product_id, verification_code_order.user_id, verification_code_order.order_id as v_order_id").
		Joins("JOIN verification_code_order ON payment_orders.order_id = verification_code_order.order_id").
		Where("payment_orders.status = ? AND payment_orders.is_callback = ?", common.OrderStatusPaid, false).
		Find(&paymentOrders).Error
	if err != nil {
		return fmt.Errorf("查询已支付订单失败: %v", err)
	}

	if len(paymentOrders) == 0 {
		sdlog.Info("没有需要处理的已支付订单")
		return nil
	}

	sdlog.Infof("找到 %d 个需要处理的已支付订单", len(paymentOrders))

	// 处理每个订单
	for _, paymentOrder := range paymentOrders {
		err = processPaymentOrder(ctx, nu, paymentOrder)
		if err != nil {
			sdlog.Errorf("处理订单失败: payment_order_id=%s, verification_order_id=%d, err=%v", paymentOrder.OrderID, paymentOrder.VOrderID, err)
			continue
		}
	}

	return nil
}

func processPaymentOrder(ctx context.Context, nu *nucl.Nucleus, paymentOrder PaymentOrderWithVerificationOrder) error {
	// 开启事务
	tx := nu.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查询产品信息
	var product model.Product
	err := tx.Where("id = ?", paymentOrder.ProductID).First(&product).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("查询产品信息失败: %v", err)
	}

	// 查询用户信息
	var user model.User
	err = tx.Where("user_id = ?", paymentOrder.UserID).First(&user).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("查询用户信息失败: %v", err)
	}

	// 如果产品等级大于用户当前等级，更新用户身份标签
	if product.Level > user.IdentityTag {
		err = tx.Model(&model.User{}).
			Where("user_id = ?", paymentOrder.UserID).
			Updates(map[string]interface{}{
				"identity_tag":  product.Level,
				"identity_name": product.LevelTag,
			}).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("更新用户身份标签失败: %v", err)
		}
		sdlog.Infof("更新用户身份标签成功: user_id=%d, new_level=%d", paymentOrder.UserID, product.Level)
	}

	// 批量生成核销码
	var codes []model.VerificationCode
	for i := 0; i < int(paymentOrder.CodeNum); i++ {
		code := model.VerificationCode{
			BuyUID:    paymentOrder.UserID,
			OrderID:   paymentOrder.VOrderID,
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
	if err = tx.Create(&codes).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("批量生成核销码失败: %v", err)
	}

	// 更新payment_orders的is_callback为true
	err = tx.Model(&model.PaymentOrder{}).
		Where("order_id = ?", paymentOrder.OrderID).
		Update("is_callback", true).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("更新支付订单回调状态失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	sdlog.Infof("订单处理成功: payment_order_id=%s, verification_order_id=%d, user_id=%d, code_num=%d",
		paymentOrder.OrderID, paymentOrder.VOrderID, paymentOrder.UserID, paymentOrder.CodeNum)

	return nil
}
