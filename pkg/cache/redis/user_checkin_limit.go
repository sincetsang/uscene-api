package redis

import (
	"context"
	"fmt"
	"time"

	"TMA/pkg/nucl"
)

const dailyPostCountKeyPattern = "user:checkin:count:%d:%s"

// AcquireDailyUserCheckinSlot 尝试为用户占用当日一次打卡额度，成功返回对应 key。
func AcquireDailyUserCheckinSlot(ctx context.Context, nu *nucl.Nucleus, userID int64, limit int64) (string, bool, error) {
	key := fmt.Sprintf(dailyPostCountKeyPattern, userID, time.Now().Format("20060102"))

	count, err := nu.RedisClient.Incr(ctx, key).Result()
	if err != nil {
		return "", false, err
	}

	if count == 1 {
		if err := expireAtNextDayZero(ctx, nu, key); err != nil {
			// 尝试清理，避免脏数据
			_, _ = nu.RedisClient.Del(ctx, key).Result()
			return "", false, err
		}
	}

	if count > limit {
		if _, err := nu.RedisClient.Decr(ctx, key).Result(); err != nil {
			return "", false, err
		}
		return "", false, nil
	}

	return key, true, nil
}

// ReleaseDailyUserCheckinSlot 释放当日占用，通常在业务失败时调用。
func ReleaseDailyUserCheckinSlot(ctx context.Context, nu *nucl.Nucleus, key string) error {
	if key == "" {
		return nil
	}

	luaScript := `
local current = redis.call("GET", KEYS[1])
if not current then
	return 0
end
if tonumber(current) <= 0 then
	redis.call("DEL", KEYS[1])
	return 0
end
return redis.call("DECR", KEYS[1])
`

	_, err := nu.RedisClient.Eval(ctx, luaScript, []string{key}).Result()
	return err
}

func expireAtNextDayZero(ctx context.Context, nu *nucl.Nucleus, key string) error {
	now := time.Now()
	nextDay := now.AddDate(0, 0, 1)
	nextZero := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, now.Location())
	expireDuration := nextZero.Sub(now)
	if expireDuration <= 0 {
		expireDuration = 24 * time.Hour
	}
	return nu.RedisClient.Expire(ctx, key, expireDuration).Err()
}
