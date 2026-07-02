package utils

import (
	"math/rand"
	"time"
)

// 使用 NewSource 和 New 创建一个本地随机数生成器，避免使用已弃用的 rand.Seed 函数
// 如果这里报错是因为重复声明，可能是在别处已经有同名函数了。
// 假设是要保留当前函数，且之前的重复声明是错误的，可直接保留此函数。
func GetRandomString(length int) string {
	var r = rand.New(rand.NewSource(time.Now().UnixNano()))
	const base = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = base[r.Intn(len(base))]
	}
	return string(result)
}
