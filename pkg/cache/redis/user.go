package redis

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"context"
	"fmt"
	"time"
)

// AcquireUserBalanceLock 获取修改操作时用户余额的分布式锁
func AcquireUserBalanceLock(ctx context.Context, nu *nucl.Nucleus, uid string) bool {

	expiration := 5 * time.Second
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	lockKey := common.Key(common.PrefixUserBalanceLock, uid)
	result, err := nu.RedisClient.SetNX(ctx, lockKey, lockValue, expiration).Result()
	if err != nil {
		return false
	}

	if !result {
		return false
	}

	go func() {
		// 直接创建一个新的 context 进行锁释放操作
		backgroundCtx := context.Background()

		select {
		case <-ctx.Done(): // 当上下文取消时释放锁
			if err = ReleaseUserBalanceLock(backgroundCtx, nu, uid, lockValue); err != nil {
				fmt.Printf("Failed to release lock for uid %s on context cancel: %v", uid, err)
				fmt.Println()
			}
		case <-time.After(expiration): // 到期自动释放锁
			if err = ReleaseUserBalanceLock(backgroundCtx, nu, uid, lockValue); err != nil {
				fmt.Printf("Failed to release lock for uid %s after expiration: %v", uid, err)
				fmt.Println()
			}
		}
	}()

	return true
}

// ReleaseUserBalanceLock 释放用户余额的分布式锁
func ReleaseUserBalanceLock(ctx context.Context, nu *nucl.Nucleus, uid, lockValue string) error {
	lockKey := common.Key(common.PrefixUserBalanceLock, uid)

	luaScript := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	_, err := nu.RedisClient.Eval(ctx, luaScript, []string{lockKey}, lockValue).Result()
	return err
}
