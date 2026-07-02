package redis

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"context"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func userClaimKey(uid int64) string {
	return common.Key("user_claim", strconv.FormatInt(uid, 10))
}

func userClaimedTotalKey(uid int64) string {
	return common.Key("user_claim_claimed", strconv.FormatInt(uid, 10))
}

// AddClaim 增加某个用户在某个 point 的 claim 值（可正可负）
func AddClaim(ctx context.Context, nu *nucl.Nucleus, uid int64, pointID string, amount float64) error {
	key := userClaimKey(uid)

	// 使用 HINCRBYFLOAT 实现累加
	return nu.RedisClient.HIncrByFloat(ctx, key, pointID, amount).Err()
}

// GetClaim 获取某个用户在某个 point 下的 claim 值
func GetClaim(ctx context.Context, nu *nucl.Nucleus, uid int64, pointID string) (float64, error) {
	key := userClaimKey(uid)

	val, err := nu.RedisClient.HGet(ctx, key, pointID).Result()
	if err == redis.Nil {
		return 0, nil // 不存在的 point，返回 0
	}
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(val, 64)
}

// GetTotalClaim 获取某个用户所有 point 的 claim 总和
func GetTotalClaim(ctx context.Context, nu *nucl.Nucleus, uid int64) (float64, error) {
	key := userClaimKey(uid)

	claims, err := nu.RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	var total float64
	for _, val := range claims {
		fv, err := strconv.ParseFloat(val, 64)
		if err != nil {
			continue
		}
		total += fv
	}
	return total, nil
}

// Claim 提现（领取某个 point 的 claim，并记录到 claimed 总值中）
func Claim(ctx context.Context, nu *nucl.Nucleus, uid int64, pointID string) (float64, error) {
	key := userClaimKey(uid)
	claimedKey := userClaimedTotalKey(uid)

	// 获取当前值
	val, err := nu.RedisClient.HGet(ctx, key, pointID).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	claimVal, err := strconv.ParseFloat(val, 64)
	if err != nil || claimVal == 0 {
		return 0, nil
	}

	// 使用 pipeline：删除该字段 + 累加到 claimed 总值
	pipe := nu.RedisClient.TxPipeline()
	pipe.HDel(ctx, key, pointID)
	pipe.IncrByFloat(ctx, claimedKey, claimVal)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return claimVal, nil
}

// GetTotalClaimed 获取用户累计领取的 claim 值
func GetTotalClaimed(ctx context.Context, nu *nucl.Nucleus, uid int64) (float64, error) {
	key := userClaimedTotalKey(uid)

	val, err := nu.RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(val, 64)
}

// ClaimAll 一次性领取用户所有 point 的 claim 值，并记录到 claimed 总值中
func ClaimAll(ctx context.Context, nu *nucl.Nucleus, uid int64) (float64, error) {
	key := userClaimKey(uid)
	claimedKey := userClaimedTotalKey(uid)

	// 获取所有 point 的 claim
	claims, err := nu.RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if len(claims) == 0 {
		return 0, nil // 无可领取
	}

	var totalClaim float64
	for _, val := range claims {
		fv, err := strconv.ParseFloat(val, 64)
		if err != nil {
			continue
		}
		totalClaim += fv
	}

	// 使用事务 pipeline：删除 hash，增加 claimed 总值
	pipe := nu.RedisClient.TxPipeline()
	pipe.Del(ctx, key)
	pipe.IncrByFloat(ctx, claimedKey, totalClaim)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return totalClaim, nil
}
