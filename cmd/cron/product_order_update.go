package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/service/db/model"
	"context"
	"fmt"
	"time"
)

const (
	productOrderUpdateLockKey = "product_order_update_lock"
	productOrderUpdateLockTTL = 10 * time.Minute
)

// 定义关联查询结果的结构体
type PaymentOrderWithProductOrder struct {
	model.PaymentOrder
	ProductID int64 `gorm:"column:product_id"`
	UserID    int64 `gorm:"column:user_id"`
	POrderID  int64 `gorm:"column:p_order_id"` // product_order的order_id
}

func productOrderUpdate(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, productOrderUpdateLockKey, lockValue, productOrderUpdateLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, productOrderUpdateLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	// 查询所有已支付但未回调的订单，关联verification_code_order表
	var paymentOrders []PaymentOrderWithProductOrder
	err = nu.DB.WithContext(ctx).
		Table("payment_orders").
		Select("payment_orders.*, product_order.product_id, product_order.user_id, product_order.order_id as p_order_id").
		Joins("JOIN product_order ON payment_orders.order_id = product_order.order_id").
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
		err = processProductOrder(ctx, nu, paymentOrder)
		if err != nil {
			sdlog.Errorf("处理订单失败: payment_order_id=%s, product_order_id=%d, err=%v", paymentOrder.OrderID, paymentOrder.POrderID, err)
			continue
		}
	}

	return nil
}

func processProductOrder(ctx context.Context, nu *nucl.Nucleus, paymentOrder PaymentOrderWithProductOrder) error {
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

	sdlog.Infof("订单处理成功: payment_order_id=%s, verification_order_id=%d, user_id=%d",
		paymentOrder.OrderID, paymentOrder.POrderID, paymentOrder.UserID)

	return nil
}
