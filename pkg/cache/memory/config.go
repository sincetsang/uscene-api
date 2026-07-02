package memory

import (
	"TMA/pkg/cache"
	"TMA/pkg/nucl"
	"TMA/service/db/model"
	"context"
	"time"
)

var ConfigMemoryCache *ConfigList

func InitConfig(nu *nucl.Nucleus, updatePeriod time.Duration) *ConfigList {
	ConfigMemoryCache = &ConfigList{}
	ConfigMemoryCache.UpdatePeriod = updatePeriod
	ConfigMemoryCache.Nu = nu
	ConfigMemoryCache.CacheName = "config"
	ConfigMemoryCache.DataGenerator = ConfigMemoryCache.QueryData
	ConfigMemoryCache.DataSetter = ConfigMemoryCache.SetData

	return ConfigMemoryCache
}

type ConfigList struct {
	memory.DefaultCache
	configListCache map[string]string
}

func (e *ConfigList) GetConfigList() map[string]string {
	e.RLock()
	defer e.RUnlock()
	return e.configListCache
}

func (e *ConfigList) QueryData(ctx context.Context) (interface{}, error) {
	configs, err := e.GetData(ctx)
	return configs, err
}

func (e *ConfigList) GetData(ctx context.Context) (map[string]string, error) {

	dataMap := make(map[string]string)

	list := []model.Config{}
	err := e.Nu.DB.WithContext(ctx).Model(&model.User{}).Find(&list).Error
	if err != nil {
		return dataMap, err
	}
	for _, item := range list {
		dataMap[item.Key] = item.Value
	}
	return dataMap, nil
}

func (e *ConfigList) SetData(data interface{}) {
	e.configListCache = data.(map[string]string)
}
