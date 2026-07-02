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
	payTimeoutLockKey = "pay_timeout_lock"
	payTimeoutLockTTL = 10 * time.Minute
)

func payTimeoutCheck(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, payTimeoutLockKey, lockValue, payTimeoutLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, payTimeoutLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	_ = payTimeoutCheckVerificationCode(ctx, nu)
	_ = payTimeoutCheckProduct(ctx, nu)
	_ = payTimeoutCheckMember(ctx, nu)

	// // 计算15分钟前的时间点
	// timeoutTime := time.Now().Add(-15 * time.Minute)

	// // 查询所有超时未支付的订单
	// var orders []model.VerificationCodeOrder
	// err = nu.DB.WithContext(ctx).
	// 	Where("status = ? AND created_at < ?", common.VerificationCodeOrderStatusUnpaid, timeoutTime).
	// 	Find(&orders).Error
	// if err != nil {
	// 	return fmt.Errorf("查询超时订单失败: %v", err)
	// }

	// if len(orders) == 0 {
	// 	return nil
	// }

	// // 批量更新订单状态为已取消
	// for _, order := range orders {
	// 	err = nu.DB.WithContext(ctx).Model(&model.VerificationCodeOrder{}).
	// 		Where("id = ?", order.ID).
	// 		Update("status", common.VerificationCodeOrderStatusCancel).Error
	// 	if err != nil {
	// 		sdlog.Errorf("[payTimeoutCheck] 取消订单失败: order_id=%d, err=%v", order.OrderID, err)
	// 		continue
	// 	}
	// 	// 更新支付订单状态为已取消
	// 	_ = nu.DB.WithContext(ctx).Model(&model.PaymentOrder{}).
	// 		Where("order_id = ?", order.OrderID).
	// 		Update("status", common.OrderStatusCancel).Error

	// 	sdlog.Infof("[payTimeoutCheck] 订单超时未支付，已自动取消: order_id=%d", order.OrderID)
	// }

	return nil
}

func payTimeoutCheckVerificationCode(ctx context.Context, nu *nucl.Nucleus) error {
	sdlog.Infof("[payTimeoutCheck] 开始检查验证码订单超时未支付")
	// 计算15分钟前的时间点
	timeoutTime := time.Now().Add(-15 * time.Minute)

	// 查询所有超时未支付的订单
	var orders []model.VerificationCodeOrder
	err := nu.DB.WithContext(ctx).
		Where("status = ? AND created_at < ?", common.VerificationCodeOrderStatusUnpaid, timeoutTime).
		Find(&orders).Error
	if err != nil {
		return fmt.Errorf("查询超时订单失败: %v", err)
	}

	if len(orders) == 0 {
		return nil
	}

	// 批量更新订单状态为已取消
	for _, order := range orders {
		err = nu.DB.WithContext(ctx).Model(&model.VerificationCodeOrder{}).
			Where("id = ?", order.ID).
			Update("status", common.VerificationCodeOrderStatusCancel).Error
		if err != nil {
			sdlog.Errorf("[payTimeoutCheck] 取消订单失败: order_id=%d, err=%v", order.OrderID, err)
			continue
		}
		// 更新支付订单状态为已取消
		_ = nu.DB.WithContext(ctx).Model(&model.PaymentOrder{}).
			Where("order_id = ?", order.OrderID).
			Update("status", common.OrderStatusCancel).Error

		sdlog.Infof("[payTimeoutCheck] 订单超时未支付，已自动取消: order_id=%d", order.OrderID)
	}
	return nil
}

func payTimeoutCheckProduct(ctx context.Context, nu *nucl.Nucleus) error {
	sdlog.Infof("[payTimeoutCheck] 开始检查商品订单超时未支付")
	// 计算15分钟前的时间点
	timeoutTime := time.Now().Add(-15 * time.Minute)

	// 查询所有超时未支付的订单
	var orders []model.ProductOrder
	err := nu.DB.WithContext(ctx).
		Where("status IN (?) AND created_at < ?", []string{common.ProductOrderStatusPendingPayment, common.ProductOrderStatusPaymentProcessing}, timeoutTime).
		Find(&orders).Error
	if err != nil {
		return fmt.Errorf("查询超时订单失败: %v", err)
	}

	sdlog.Infof("[payTimeoutCheck] 查询到超时未支付的商品订单数量: %d", len(orders))
	if len(orders) == 0 {
		return nil
	}

	// 批量更新订单状态为已取消
	for _, order := range orders {
		err = nu.DB.WithContext(ctx).Model(&model.ProductOrder{}).
			Where("id = ?", order.ID).
			Update("status", common.ProductOrderStatusCancelled).Error
		if err != nil {
			sdlog.Errorf("[payTimeoutCheck] 取消订单失败: order_id=%d, err=%v", order.OrderID, err)
			continue
		}
		// 更新支付订单状态为已取消
		_ = nu.DB.WithContext(ctx).Model(&model.PaymentOrder{}).
			Where("order_id = ?", order.OrderID).
			Update("status", common.OrderStatusCancel).Error

		sdlog.Infof("[payTimeoutCheck] 订单超时未支付，已自动取消: order_id=%d", order.OrderID)
	}
	return nil
}

func payTimeoutCheckMember(ctx context.Context, nu *nucl.Nucleus) error {
	sdlog.Infof("[payTimeoutCheck] 开始检查会员订单超时未支付")
	// 计算15分钟前的时间点
	timeoutTime := time.Now().Add(-15 * time.Minute)

	// 查询所有超时未支付的订单
	var orders []model.MemberOrder
	err := nu.DB.WithContext(ctx).
		Where("status in (?) AND created_at < ?", []string{common.MemberOrderStatusUnpaid, common.VerificationCodeOrderStatusUnpaid}, timeoutTime).
		Find(&orders).Error
	if err != nil {
		return fmt.Errorf("查询超时订单失败: %v", err)
	}

	sdlog.Infof("[payTimeoutCheck] 查询到超时未支付的会员订单数量: %d", len(orders))
	if len(orders) == 0 {
		return nil
	}

	// 批量更新订单状态为已取消
	for _, order := range orders {
		err = nu.DB.WithContext(ctx).Model(&model.MemberOrder{}).
			Where("id = ?", order.ID).
			Update("status", common.VerificationCodeOrderStatusCancel).Error
		if err != nil {
			sdlog.Errorf("[payTimeoutCheck] 取消订单失败: order_id=%d, err=%v", order.OrderID, err)
			continue
		}
		sdlog.Infof("[payTimeoutCheck] 订单超时未支付，已自动取消: order_id=%d", order.OrderID)
		_ = nu.DB.WithContext(ctx).Model(&model.PaymentOrder{}).
			Where("order_id = ?", order.OrderID).
			Update("status", common.OrderStatusCancel).Error
	}
	return nil
}
