package redis

import (
	"TMA/pkg/nucl"
	"context"
	"fmt"
	"time"
)

// TryLock 尝试获取分布式锁
func TryLock(ctx context.Context, nu *nucl.Nucleus, key, lockValue string, ttl time.Duration) (bool, error) {
	result, err := nu.RedisClient.SetNX(ctx, key, lockValue, ttl).Result()
	if err != nil {
		return false, err
	}

	if !result {
		return false, nil
	}

	// 设置锁的自动释放
	go func() {
		backgroundCtx := context.Background()
		select {
		case <-ctx.Done(): // 当上下文取消时释放锁
			if err = Unlock(backgroundCtx, nu, key, lockValue); err != nil {
				fmt.Printf("Failed to release lock for key %s on context cancel: %v", key, err)
			}
		case <-time.After(ttl): // 到期自动释放锁
			if err = Unlock(backgroundCtx, nu, key, lockValue); err != nil {
				fmt.Printf("Failed to release lock for key %s after expiration: %v", key, err)
			}
		}
	}()

	return true, nil
}

// Unlock 释放分布式锁
func Unlock(ctx context.Context, nu *nucl.Nucleus, key, lockValue string) error {
	luaScript := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	_, err := nu.RedisClient.Eval(ctx, luaScript, []string{key}, lockValue).Result()
	return err
}
