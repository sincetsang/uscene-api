package redis

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"context"
	"fmt"
	"strconv"
)

// SetUserAsNew 设置用户为新用户（无过期时间）
func SetUserAsNew(ctx context.Context, nu *nucl.Nucleus, uid int64) error {
	// 定义用户的 Redis 键
	userKey := common.Key(common.PrefixNewUser, strconv.FormatInt(uid, 10))

	// 将用户设置为新用户，不设置过期时间
	err := nu.RedisClient.Set(ctx, userKey, "new_user", 0).Err()
	if err != nil {
		fmt.Printf("Failed to set new user for uid %d: %v", uid, err)
		return err
	}

	return nil
}

// IsNewUser 判断是否为新用户
func IsNewUser(ctx context.Context, nu *nucl.Nucleus, uid int64) (bool, error) {
	// 定义用户的 Redis 键
	userKey := common.Key(common.PrefixNewUser, strconv.FormatInt(uid, 10))

	// 检查用户是否存在于 Redis 中
	exists, err := nu.RedisClient.Exists(ctx, userKey).Result()
	if err != nil {
		fmt.Printf("Failed to check user existence for uid %d: %v", uid, err)
		return false, err
	}
	if exists == 0 {
		return false, nil
	}
	return true, nil
}

// RemoveUserStatus 删除用户的新用户状态
func RemoveUserStatus(ctx context.Context, nu *nucl.Nucleus, uid int64) error {
	// 定义用户的 Redis 键
	userKey := common.Key(common.PrefixNewUser, strconv.FormatInt(uid, 10))

	// 删除 Redis 中的用户记录
	err := nu.RedisClient.Del(ctx, userKey).Err()
	if err != nil {
		fmt.Printf("Failed to remove user status for uid %d: %v", uid, err)
		return err
	}

	return nil
}
