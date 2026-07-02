package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/pkg/web/webapi"
	"TMA/service"
)

func spotListV4(ec *middleware.AppRequestContext) error {
	radius := ec.QueryParams().Get("radius")
	radiusInt := sdparse.Int64Def(radius, 0)
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	isOwner := ec.QueryParams().Get("is_owner")
	keyword := ec.QueryParams().Get("keyword")
	sortBy := ec.QueryParams().Get("sort_by") // 排序方式：distance(距离)、reward(奖励)、time(时间)，默认distance
	hasAdvertisement := ec.QueryParams().Get("has_advertisement")
	hasAdvertisementInt := sdparse.Int64Def(hasAdvertisement, 0)
	category := ec.QueryParams().Get("category")
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 30)
	if pageSizeInt > 30 {
		pageSizeInt = 30
	}
	userId := ec.AuthData.User.ID
	info, err := service.SpotListV4(ec.Request().Context(), ec.Nu, userId, radiusInt, latitude, longitude, isOwnerInt, keyword, category, sortBy, hasAdvertisementInt, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("[spotListV2] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

func spotListGrid(ec *middleware.AppRequestContext) error {
	// 获取地图边界参数
	swLat := ec.QueryParams().Get("sw_lat") // 西南角纬度
	swLng := ec.QueryParams().Get("sw_lng") // 西南角经度
	neLat := ec.QueryParams().Get("ne_lat") // 东北角纬度
	neLng := ec.QueryParams().Get("ne_lng") // 东北角经度
	keyword := ec.QueryParams().Get("keyword")
	category := ec.QueryParams().Get("category")
	isOwner := ec.QueryParams().Get("is_owner")
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	userId := ec.AuthData.User.ID

	// 参数验证
	if swLat == "" || swLng == "" || neLat == "" || neLng == "" {
		return webapi.Error(common.ErrLocationLatLonErr).Render(ec)
	}

	// 转换参数类型
	swLatFloat := sdparse.Float64Def(swLat, 0)
	swLngFloat := sdparse.Float64Def(swLng, 0)
	neLatFloat := sdparse.Float64Def(neLat, 0)
	neLngFloat := sdparse.Float64Def(neLng, 0)

	// 调用服务层方法获取网格化数据
	info, err := service.SpotListGrid(ec.Request().Context(), ec.Nu, userId, swLatFloat, swLngFloat, neLatFloat, neLngFloat, isOwnerInt, keyword, category)
	if err != nil {
		sdlog.Errorf("[spotListGrid] 查询网格化地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(info).Render(ec)
}

func spotListPithy(ec *middleware.AppRequestContext) error {
	// 获取地图边界参数
	swLat := ec.QueryParams().Get("sw_lat") // 西南角纬度
	swLng := ec.QueryParams().Get("sw_lng") // 西南角经度
	neLat := ec.QueryParams().Get("ne_lat") // 东北角纬度
	neLng := ec.QueryParams().Get("ne_lng") // 东北角经度
	keyword := ec.QueryParams().Get("keyword")
	category := ec.QueryParams().Get("category")
	isOwner := ec.QueryParams().Get("is_owner")
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 20)
	if pageSizeInt > 20 {
		pageSizeInt = 20
	}
	isOwnerInt := sdparse.Int64Def(isOwner, 0)
	userId := ec.AuthData.User.ID

	// 参数验证
	if swLat == "" || swLng == "" || neLat == "" || neLng == "" {
		return webapi.Error(common.ErrLocationLatLonErr).Render(ec)
	}

	// 转换参数类型
	swLatFloat := sdparse.Float64Def(swLat, 0)
	swLngFloat := sdparse.Float64Def(swLng, 0)
	neLatFloat := sdparse.Float64Def(neLat, 0)
	neLngFloat := sdparse.Float64Def(neLng, 0)

	// 调用服务层方法获取网格化数据
	info, err := service.SpotListPithy(ec.Request().Context(), ec.Nu, userId, swLatFloat, swLngFloat, neLatFloat, neLngFloat, isOwnerInt, keyword, category, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("[spotListPithy] 查询网格化地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(info).Render(ec)
}

func spotListByDistance(ec *middleware.AppRequestContext) error {
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	category := ec.QueryParams().Get("category")
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 30)
	if pageSizeInt > 30 {
		pageSizeInt = 30
	}
	userId := ec.AuthData.User.ID
	info, err := service.SpotListByDistance(ec.Request().Context(), ec.Nu, userId, latitude, longitude, category, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("[spotListByDistance] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}

func spotListByDistanceV2(ec *middleware.AppRequestContext) error {
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	category := ec.QueryParams().Get("category")
	page := ec.QueryParams().Get("page")
	pageInt := sdparse.Int64Def(page, 1)
	pageSize := ec.QueryParams().Get("page_size")
	pageSizeInt := sdparse.Int64Def(pageSize, 30)
	userId := ec.AuthData.User.ID
	info, err := service.SpotListByDistanceV2(ec.Request().Context(), ec.Nu, userId, latitude, longitude, category, pageInt, pageSizeInt)
	if err != nil {
		sdlog.Errorf("[spotListByDistanceV2] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(info).Render(ec)
}
