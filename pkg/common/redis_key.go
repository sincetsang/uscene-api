package common

import (
	"fmt"
	"strings"
)

type RedisPrefix string

const (
	keySplitter = ":"
	// HFieldSplitter hash类型的redis缓存，按读取需求，多个属性可以放到一个field里，用双下划线分隔：__
	HFieldSplitter = "__"
)

const (
	PrefixUserBalanceLock   = RedisPrefix("00") // 操作用户余额时的锁
	PrefixNewUser           = RedisPrefix("01") // 新用户
	PrefixChatPopularity    = RedisPrefix("10") // 群组热度
	PrefixCheckInPointLock  = RedisPrefix("20")
	PrefixWalletAddressCode = RedisPrefix("30")
)

func Key(prefix RedisPrefix, key string, otherParts ...string) string {
	oP := ""
	if len(otherParts) > 0 {
		oP = keySplitter + strings.Join(otherParts, keySplitter)
	}
	return fmt.Sprintf("%s%s%s%s", prefix, keySplitter, key, oP)
}
func JoinHFieldValue(values ...string) string {
	return strings.Join(values, HFieldSplitter)
}

func SplitHFieldValue(value string) []string {
	return strings.Split(value, HFieldSplitter)
}
