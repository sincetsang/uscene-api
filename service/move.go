package service

import (
	"math"
)

// LocationObject 定义前端上报的位置信息结构体
type LocationObject struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Altitude  *float64 `json:"altitude,omitempty"` // 可选字段，高度
	Timestamp int64    `json:"timestamp"`          // 时间戳（毫秒）
}

// Haversine 公式计算球面距离（单位：米）
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // 地球半径，单位：米
	lat1, lon1, lat2, lon2 = toRadians(lat1), toRadians(lon1), toRadians(lat2), toRadians(lon2)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// CalculateMoveTime 计算运动时间
func CalculateMoveTime(locations []LocationObject) int {
	moveSeconds := 0

	// 遍历相邻的坐标点
	for i := 1; i < len(locations); i++ {
		prev := locations[i-1]
		curr := locations[i]

		// 计算时间差（秒）
		timeDiff := float64(curr.Timestamp-prev.Timestamp) / 1000.0
		if timeDiff <= 0 {
			continue // 避免时间异常
		}

		// 计算水平距离（基于经纬度）
		horizontalDistance := haversine(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)

		// 计算高度变化
		altitudeDiff := 0.0
		if prev.Altitude != nil && curr.Altitude != nil {
			altitudeDiff = *curr.Altitude - *prev.Altitude
		}

		// 计算三维欧几里得距离
		totalDistance := math.Sqrt(horizontalDistance*horizontalDistance + altitudeDiff*altitudeDiff)

		// 计算速度
		speed := totalDistance / timeDiff // 单位：m/s

		// 如果速度大于 0.5m/s，则累加运动时间
		if speed > 0.5 {
			moveSeconds += int(timeDiff)
		}
	}
	return moveSeconds
}
