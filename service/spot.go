package service

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CheckInInfoV3 struct {
	UserCheckInLimit int32                  `json:"user_check_in_limit"` // 用户打卡限制值
	UserCheckInCount int32                  `json:"user_check_in_count"` // 今日已打卡数
	TotalCount       int64                  `json:"total_count"`         // 地标总数
	Points           []CheckInPointStructV3 `json:"points"`
}

type CheckInPointStructV3 struct {
	Distance           float64                   `json:"distance"`              // 与用户的距离
	CheckedIn          bool                      `json:"checked_in"`            // 是否已打卡
	Liked              bool                      `json:"liked"`                 // 是否已点赞
	CanCheckIn         bool                      `json:"can_check_in"`          // 是否可以打卡
	CheckInAward       string                    `json:"check_in_award"`        // 打卡奖励
	CheckInMaxDistance float64                   `json:"check_in_max_distance"` // 地标最远打卡距离
	Point              *db.CheckInPointWithOwner `json:"point"`                 // 地标详情
	CategoryImage      string                    `json:"category_image"`        // 地标分类图片
}

type ByDistanceV3 []CheckInPointStructV3

func (a ByDistanceV3) Len() int           { return len(a) }
func (a ByDistanceV3) Less(i, j int) bool { return a[i].Distance < a[j].Distance }
func (a ByDistanceV3) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func SpotList(ctx context.Context, nu *nucl.Nucleus, uid int64, radius int64, latitudeStr string, longitudeStr string, isOwner int64, keyword string, like int64) (*CheckInInfoV3, error) {

	if latitudeStr == "" || longitudeStr == "" {
		return nil, common.ErrLocationLatLonErr
	}
	latitude := sdparse.Float64Def(latitudeStr, 0.0)
	longitude := sdparse.Float64Def(longitudeStr, 0.0)
	userInfo, err := db.UserInfoById(nu.DB.WithContext(ctx), uid)
	if err != nil {
		return nil, common.ErrUserNotExist
	}

	userLevelConfig, errL := db.UserLevelInfoById(nu.DB.WithContext(ctx), userInfo.Level)
	if errL != nil {
		return nil, common.ErrService
	}

	spotLevelConfigDic, errSlc := spotLevelDic(ctx, nu)
	if errSlc != nil {
		return nil, common.ErrService
	}

	result := CheckInInfoV3{}
	result.UserCheckInLimit = userLevelConfig.CheckinLimit

	visableDistanceStr := db.GetConfigByKey(nu.DB.WithContext(ctx), common.ConfigCheckPointVisibleDistance, "10000")
	visableDistance_, _ := sdparse.Float64(*visableDistanceStr)

	var checkPoints []*db.CheckInPointWithOwner
	distance := float64(radius)
	if radius == 0 {
		distance = 80000000
	} else if radius == -1 {
		distance = visableDistance_
	}

	latMin, latMax, lonMin, lonMax := getBoundingBox(latitude, longitude, distance/1000)
	sdlog.Infof("latMin: %f, latMax: %f, lonMin: %f, lonMax: %f", latMin, latMax, lonMin, lonMax)
	tmpOwnerId := int64(0)
	if isOwner == 1 {
		tmpOwnerId = uid
	}
	// if like != 0 {
	like = uid
	// }
	checkPoints_, errC := db.GetAllValidCheckPointsByLaLo(nu.DB.WithContext(ctx), latMin, latMax, lonMin, lonMax, tmpOwnerId, keyword, like)
	if errC != nil {
		return &result, errC
	}
	checkPoints = checkPoints_

	todayDate := time.Now().Format("20060102")
	todayCheckRecords, errU := db.GetUserCheckPointsByUserIdAndDate(nu.DB.WithContext(ctx), uid, todayDate)
	if errU != nil {
		return &result, errU
	}
	var checkRecordsMap = map[int64]bool{}
	for _, ch := range todayCheckRecords {
		checkRecordsMap[ch.PointID] = true
	}

	var pointIdArr = make([]int64, 0)
	pointsArr := []CheckInPointStructV3{}
	for _, point := range checkPoints {
		pointIdArr = append(pointIdArr, point.ID)
		pointDistance := float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude))
		if pointDistance < distance {
			redis.AddClaim(ctx, nu, point.OwnerID, fmt.Sprint(point.ID), 1)
			claim, _ := redis.GetClaim(ctx, nu, point.OwnerID, fmt.Sprint(point.ID))
			point.Claim = claim
			checkInfo := CheckInPointStructV3{
				Distance:  float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude)),
				CheckedIn: checkRecordsMap[point.ID],
				Point:     point,
				Liked:     point.Liked,
			}
			pointsArr = append(pointsArr, checkInfo)
		}
	}
	sort.Sort(ByDistanceV3(pointsArr))

	checkCountArr, errCC := db.SpotUserCheckedByIds(nu.DB.WithContext(ctx), uid, pointIdArr)
	if errCC != nil {
		return nil, common.ErrUnknown
	}
	var checkCountDic = make(map[int64]db.UserSpotChecked, 0)
	for _, item := range checkCountArr {
		checkCountDic[item.PointId] = item
	}

	checkRewardPercentageStr := db.GetConfigByKey(nu.DB.WithContext(ctx), common.ConfigCheckPointRewardPercentage, "0.05")
	checkRewardPercentage, _ := sdparse.Float64(*checkRewardPercentageStr)

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err = nu.DB.WithContext(ctx).Model(&model.SpotCategory{}).Find(&categories).Error
	if err != nil {
		return nil, common.ErrService
	}
	categoryMap := make(map[string]*model.SpotCategory)
	for _, cat := range categories {
		categoryMap[cat.Type] = cat
	}

	finalArr := []CheckInPointStructV3{}
	for _, point := range pointsArr {
		_, ok := checkCountDic[point.Point.ID]
		if !ok && userLevelConfig.CheckinLimit > int32(len(todayCheckRecords)) && point.Point.Balance > 1 && point.CheckInMaxDistance >= point.Distance {
			point.CanCheckIn = true
		} else {
			point.CanCheckIn = false
		}

		point.CheckInAward = currencyValue(common.ConfigPlatformCoinName, point.Point.Balance*checkRewardPercentage)
		point.CheckInMaxDistance = float64(spotLevelConfigDic[int64(point.Point.Level)].CheckinRadius)
		// 设置分类图片
		if cat, exists := categoryMap[point.Point.Category]; exists {
			lan := common.GetRequestLanguage()
			if lan == common.LanguageZh {
				point.Point.Category = cat.LabelZh
			} else if lan == common.LanguageZhTw {
				point.Point.Category = cat.LabelZhTw
			} else if lan == common.LanguageRu {
				point.Point.Category = cat.LabelRu
			}
			point.CategoryImage = cat.ImageURL
		}
		finalArr = append(finalArr, point)
	}
	result.UserCheckInCount = int32(len(todayCheckRecords))
	result.Points = finalArr
	result.TotalCount = int64(len(finalArr))
	return &result, nil
}

func spotLevelDic(ctx context.Context, nu *nucl.Nucleus) (map[int64]*model.SpotLevelConfig, error) {
	configs, err := db.GetSpotLevelAll(nu.DB.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*model.SpotLevelConfig, 0)
	for _, config := range configs {
		result[int64(config.Level)] = config
	}
	return result, nil
}

type CheckInResponse struct {
	PointId int64  `json:"point_id"`
	Award   string `json:"award"`
}

func CheckInAtPoint(ctx context.Context, nu *nucl.Nucleus, uid int64, pointId int64) (*CheckInResponse, error) {

	userInfo, err := db.UserInfoById(nu.DB.WithContext(ctx), uid)
	if err != nil {
		return nil, common.ErrUnknown
	}

	spot, errCheck := db.GetValidCheckPointById(nu.DB.WithContext(ctx), pointId)
	if err == gorm.ErrRecordNotFound {
		return nil, common.ErrCheckInPointNotExist
	}

	if errCheck != nil {
		return nil, common.ErrCheckInFailed
	}

	// 打卡距离判断
	distance := calculateDistance(userInfo.Latitude, userInfo.Longitude, spot.Latitude, spot.Longitude)
	spotLevelConfig, errL := db.GetSpotLevelInfoByLevel(nu.DB.WithContext(ctx), int64(spot.Level))
	if errL != nil {
		return nil, common.ErrService
	}

	if spot.Balance < 1 {
		return nil, common.ErrSpotBalanceNotCanCheckIn
	}

	if distance > float64(spotLevelConfig.CheckinRadius) {
		//errMsg := fmt.Sprintf(`The distance from you to the target is over %.0f km. Please verify.`, float64(spotLevelConfig.CheckInDistance)/1000)
		errMsg := fmt.Sprintf(`The distance from you to the target is over %.0f m. Please verify.`, float64(spotLevelConfig.CheckinRadius))
		return nil, sderr.Sentinel(errMsg)
	}

	todayDate := time.Now().Format("20060102")
	todayCheckRecords, errU := db.GetUserCheckPointsByUserIdAndDate(nu.DB.WithContext(ctx), uid, todayDate)
	if errU != nil {
		return nil, common.ErrCheckInFailed
	}
	// 已经check-in
	checkedIn := userCheckedIn(pointId, todayCheckRecords)
	if checkedIn {
		return nil, common.ErrAllreadyCheckedIn
	}

	userLevelConfig, errL := db.UserLevelInfoById(nu.DB.WithContext(ctx), userInfo.Level)
	if errL != nil {
		return nil, common.ErrUnknown
	}

	// 打卡次数限制
	limit := userLevelConfig.CheckinLimit
	if len(todayCheckRecords) >= int(limit) {
		return nil, common.ErrCheckInOverLimit
	}

	checkRewardPercentageStr := db.GetConfigByKey(nu.DB.WithContext(ctx), common.ConfigCheckPointRewardPercentage, "0.05")
	checkRewardPercentage, _ := sdparse.Float64(*checkRewardPercentageStr)
	checkRewardPercentage = math.Floor(spot.Balance*checkRewardPercentage*100) / 100

	errUpdate := nu.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := model.UserCheckRecord{
			PointID: pointId,
			Day:     todayDate,
			UserID:  uid,
		}
		err = tx.Model(&model.UserCheckRecord{}).Create(&record).Error
		if err != nil {
			return err
		}

		addBalance := model.User{
			Balance: userInfo.Balance + checkRewardPercentage,
		}
		errUser := tx.Model(&model.User{}).Where("user_id = ?", uid).Updates(&addBalance).Error
		if err != nil {
			return errUser
		}
		err = tx.Model(&model.CheckInPoint{}).Where("id = ?", spot.ID).Updates(&model.CheckInPoint{
			TotalCheckedIn: spot.TotalCheckedIn + 1,
		}).Error
		if err != nil {
			return err
		}
		assetDetail := model.UserAssetDetail{
			UserID:    uid,
			Category:  common.ConfigAssetTypeByCheckIn,
			RelatedID: pointId,
			Currency:  common.ConfigPlatformCoinName,
			Rewards:   checkRewardPercentage,
		}
		err = tx.Model(&model.UserAssetDetail{}).Create(&assetDetail).Error
		if err != nil {
			return err
		}
		err = tx.Model(&model.CheckInPoint{}).Where("id = ?", pointId).UpdateColumn("balance", gorm.Expr("balance - ?", checkRewardPercentage)).Error
		if err != nil {
			return err
		}

		// TODO 邀请奖励

		return err
	})

	if errUpdate != nil {
		return nil, common.ErrCheckInFailed
	}
	result := CheckInResponse{
		PointId: spot.ID,
		Award:   fmt.Sprintf("%.2f", checkRewardPercentage),
	}
	return &result, nil
}

const earthRadius = 6371.0

// toRadians 将角度转换为弧度
func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// toDegrees 将弧度转换为角度
func toDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

// getBoundingBox 计算给定距离内的经纬度边界框
func getBoundingBox(lat, lon, distanceKm float64) (float64, float64, float64, float64) {
	// 计算纬度边界
	latMin := lat - toDegrees(distanceKm/earthRadius)
	latMax := lat + toDegrees(distanceKm/earthRadius)

	// 计算当前纬度上的地球半径
	radiusAtLat := earthRadius * math.Cos(toRadians(lat))

	// 计算经度边界
	lonMin := lon - toDegrees(distanceKm/radiusAtLat)
	lonMax := lon + toDegrees(distanceKm/radiusAtLat)

	return latMin, latMax, lonMin, lonMax
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Radius of Earth in kilometers

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c

	return distance * 1000
}

func currencyValue(currency string, amount float64) string {
	var str = ""
	if strings.ToLower(currency) == "ton" {
		str = strconv.FormatFloat(amount/1e9, 'f', -1, 64)
	} else if strings.ToLower(currency) == "usdt" {
		str = strconv.FormatFloat(amount/1e6, 'f', -1, 64)
	} else if strings.ToLower(currency) == common.ConfigPlatformCoinName {
		tmpAmount := math.Floor(amount*100) / 100
		str = strconv.FormatFloat(tmpAmount, 'f', -1, 64)
	}
	return str
}

func userCheckedIn(pointId int64, UserRecords []*model.UserCheckRecord) bool {
	for _, record := range UserRecords {
		if pointId == record.PointID {
			return true
		}
	}
	return false
}

// CheckAddSpotPower 创建新的打卡点时鉴权
func CheckAddSpotPower(ctx context.Context, nu *nucl.Nucleus, uid int64, latitude, longitude float64) error {

	// 当前位置是否可以创建打卡点,打卡区域不能和其他打卡点重叠
	var checkPoints []*db.CheckInPointWithOwner
	distanceStr := db.GetConfigByKey(nu.DB.WithContext(ctx), common.ConfigCheckPointCheckInDistance, "1000")
	includeMaxDistance, _ := sdparse.Float64(*distanceStr)

	latMin, latMax, lonMin, lonMax := getBoundingBox(latitude, longitude, includeMaxDistance/1000)
	checkPoints, err := db.GetAllValidCheckPointsByLaLo(nu.DB.WithContext(ctx), latMin, latMax, lonMin, lonMax, 0, "", 0)
	if err != nil {
		sdlog.Warnf("get GetAllValidCheckPointsByLaLo fail,err:%v", err)
		return common.ErrService
	}
	if len(checkPoints) > 0 {
		for _, point := range checkPoints {
			pointDistance := float64(calculateDistance(latitude, longitude, point.Latitude, point.Longitude))
			if includeMaxDistance >= pointDistance {
				return common.ErrSpotOverlap
			}
		}
	}
	return nil
}

// 异步检查 spot 的图片状态
func AsyncCheckSpotImageStatus(ctx context.Context, n *nucl.Nucleus, spotID int64) {
	// 创建一个新的 context，设置超时时间
	checkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 等待一段时间，让图片审核有足够时间完成
	select {
	case <-time.After(10 * time.Second):
		// 继续执行
	case <-checkCtx.Done():
		sdlog.Errorf("[AsyncCheckSpotImageStatus] 检查超时: %v", checkCtx.Err())
		return
	}

	// 最多重试 6 次，每次间隔 10 秒
	for i := 0; i < 18; i++ {
		// 检查 spot 的图片状态
		var approvedCount int64
		err := n.DB.WithContext(checkCtx).
			Model(&model.CheckInPointImage{}).
			Where("check_in_point_id = ? AND status = ?", spotID, common.UserImageStatusApproved).
			Count(&approvedCount).Error
		if err != nil {
			sdlog.Errorf("[AsyncCheckSpotImageStatus] 检查图片状态失败: %v", err)
			return
		}

		// 如果有一张图片审核通过，则更新 spot 状态并返回
		if approvedCount > 0 {
			err = n.DB.WithContext(checkCtx).
				Model(&model.CheckInPoint{}).
				Where("id = ?", spotID).
				Update("audit_status", common.SpotAuditStatusApproved).Error
			if err != nil {
				sdlog.Errorf("[AsyncCheckSpotImageStatus] 更新 spot 状态失败: %v", err)
			}
			sdlog.Infof("[AsyncCheckSpotImageStatus] spot %d 的图片审核通过", spotID)
			return
		}

		// 等待一段时间后重试
		select {
		case <-time.After(10 * time.Second):
			// 继续下一次检查
		case <-checkCtx.Done():
			sdlog.Errorf("[AsyncCheckSpotImageStatus] 检查超时: %v", checkCtx.Err())
			return
		}
	}

	// 如果所有重试都完成，但还没有审核通过的图片
	sdlog.Infof("[AsyncCheckSpotImageStatus] spot %d 的图片审核未完成", spotID)
}
