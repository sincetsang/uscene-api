package redis

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"context"
	"fmt"
	"time"
)

// GetCheckinPointLock 获取修改操作时用户余额的分布式锁
func GetCheckinPointLock(ctx context.Context, nu *nucl.Nucleus, uid int64, pointId string) bool {

	expiration := 10 * time.Second
	lockValue := fmt.Sprintf("%d", uid)
	lockKey := common.Key(common.PrefixCheckInPointLock, pointId)
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
			if err = DeleteCheckinPointLock(backgroundCtx, nu, pointId, lockValue); err != nil {
				fmt.Printf("Failed to release lock for point_id %s on context cancel: %v", pointId, err)
				fmt.Println()
			}
		case <-time.After(expiration): // 到期自动释放锁
			if err = DeleteCheckinPointLock(backgroundCtx, nu, pointId, lockValue); err != nil {
				fmt.Printf("Failed to release lock for point_id %s after expiration: %v", pointId, err)
				fmt.Println()
			}
		}
	}()

	return true
}

// DeleteCheckinPointLock 释放用户余额的分布式锁
func DeleteCheckinPointLock(ctx context.Context, nu *nucl.Nucleus, pointId, lockValue string) error {
	lockKey := common.Key(common.PrefixCheckInPointLock, pointId)

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
