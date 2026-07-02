package memory

import (
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"context"
	"time"
)

// MemoryCache 内存缓存
type MemoryCache interface {
	// Init 初始化数据
	Init(ctx context.Context) error
	// Update 更新数据
	Update(ctx context.Context) error
	// Name 内存缓存标识
	Name() string
	// Timer 定期更新时间，返回时长小于0则不定期更新
	Timer() time.Duration
}

// 所有内存缓存示例
var cacheInstances map[string]MemoryCache

func RegisterMemoryCache(cacheInstance ...MemoryCache) {
	if cacheInstances == nil {
		cacheInstances = make(map[string]MemoryCache)
	}
	for _, instance := range cacheInstance {
		cacheInstances[instance.Name()] = instance
	}
}

// InitMemoryCache 初始化缓存数据
func InitMemoryCache(ctx context.Context, nu *nucl.Nucleus) error {

	for name, instance := range cacheInstances {
		sdlog.WithField("name", name).Info("Init cache")
		nowTime := time.Now()
		err := instance.Init(ctx)
		if err != nil {
			sdlog.WithField("elapse", time.Now().Sub(nowTime)).
				WithField("name", instance.Name()).
				WithError(err).
				Error("update cache failed")
			return err
		} else {
			sdlog.WithField("elapse", time.Now().Sub(nowTime)).
				WithField("name", instance.Name()).
				Info("update cache success")
		}
	}
	UpdateMemoryCache()
	return nil
}

func updateCache(instance MemoryCache) {
	ticker := time.NewTicker(instance.Timer())
	for {
		select {
		case <-ticker.C:
			sdlog.WithField("name", instance.Name()).Info("start update cache")
			nowTime := time.Now()
			err := instance.Update(context.Background())
			if err != nil {
				sdlog.WithField("elapse", time.Now().Sub(nowTime)).
					WithField("name", instance.Name()).
					WithError(err).
					Error("update cache failed")
			} else {
				sdlog.WithField("elapse", time.Now().Sub(nowTime)).
					WithField("name", instance.Name()).
					Info("update cache success")
			}
		}
	}
}

// UpdateMemoryCache 启动协程，开始定期更新缓存
func UpdateMemoryCache() {
	for _, instance := range cacheInstances {
		if instance.Timer() > 0 {
			go updateCache(instance)
		}
	}
}
