package main

import (
	memory "TMA/pkg/cache"
	memory2 "TMA/pkg/cache/memory"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"context"
	"time"
)

// InitMemoryCache 内存缓存
func InitMemoryCache(nu *nucl.Nucleus) {
	// 注册
	memory.RegisterMemoryCache(
		memory2.InitCategory(nu, time.Minute*1),
	)
	// 初始化
	err := memory.InitMemoryCache(context.Background(), nu)
	if err != nil {
		sdlog.WithError(err).Error("insert cache failed")
		panic(err)
	}
}
