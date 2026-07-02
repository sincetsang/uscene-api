package memory

import (
	memory "TMA/pkg/cache"
	"TMA/pkg/nucl"
	"TMA/service/db/model"
	"context"
	"time"
)

var CategoryMemoryCache *CategoryList

func InitCategory(nu *nucl.Nucleus, updatePeriod time.Duration) *CategoryList {
	CategoryMemoryCache = &CategoryList{}
	CategoryMemoryCache.UpdatePeriod = updatePeriod
	CategoryMemoryCache.Nu = nu
	CategoryMemoryCache.CacheName = "category"
	CategoryMemoryCache.DataGenerator = CategoryMemoryCache.QueryData
	CategoryMemoryCache.DataSetter = CategoryMemoryCache.SetData

	return CategoryMemoryCache
}

type CategoryList struct {
	memory.DefaultCache
	categoryListCache map[string]model.SpotCategory
}

func (e *CategoryList) GetCategoryList() map[string]model.SpotCategory {
	e.RLock()
	defer e.RUnlock()
	return e.categoryListCache
}

func (e *CategoryList) QueryData(ctx context.Context) (interface{}, error) {
	categories, err := e.GetData(ctx)
	return categories, err
}

func (e *CategoryList) GetData(ctx context.Context) (map[string]model.SpotCategory, error) {

	dataMap := make(map[string]model.SpotCategory)

	list := []model.SpotCategory{}
	err := e.Nu.DB.WithContext(ctx).Model(&model.SpotCategory{}).Find(&list).Error
	if err != nil {
		return dataMap, err
	}
	for _, item := range list {
		dataMap[item.Type] = item
		dataMap[item.Label] = item
		dataMap[item.LabelZh] = item
		dataMap[item.LabelZhTw] = item
	}
	return dataMap, nil
}

func (e *CategoryList) SetData(data interface{}) {
	e.categoryListCache = data.(map[string]model.SpotCategory)
}
