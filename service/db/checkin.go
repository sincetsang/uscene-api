package db

import (
	"TMA/service/db/model"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type TagsStruct struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// 给出checkIn id数组，返回checkin tags map[checkin_id][]TagsStruct
func GetCheckinTagsMap(tx *gorm.DB, checkinIds []int64) (map[int64][]TagsStruct, error) {
	var tagRelations []model.SupermapCheckintagrelation
	err := tx.Model(&model.SupermapCheckintagrelation{}).Where("check_in_id IN (?)", checkinIds).Find(&tagRelations).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make(map[int64][]TagsStruct), nil
		}
		return nil, err
	}
	var tagIds []int64
	for _, tagRelation := range tagRelations {
		tagIds = append(tagIds, tagRelation.CheckInTagID)
	}
	var tagConfig []model.SupermapCheckintag
	err = tx.Model(&model.SupermapCheckintag{}).Find(&tagConfig).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make(map[int64][]TagsStruct), nil
		}
		return nil, err
	}

	var tagConfigMap map[int64]model.SupermapCheckintag = make(map[int64]model.SupermapCheckintag)
	for _, tag := range tagConfig {
		tagConfigMap[tag.ID] = tag
	}

	var tagMap map[int64][]TagsStruct = make(map[int64][]TagsStruct)
	for _, tagRelation := range tagRelations {
		if checkinTag, ok := tagConfigMap[tagRelation.CheckInTagID]; ok {
			tagMap[tagRelation.CheckInID] = append(tagMap[tagRelation.CheckInID], TagsStruct{
				ID:   tagRelation.CheckInTagID,
				Name: checkinTag.Name,
			})
		}
	}
	return tagMap, nil
}

// 给出checkIn id数组，返回checkin images map[checkin_id][]SupermapCheckinimage
func GetCheckinImagesMap(tx *gorm.DB, checkinIds []int64, status []int32) (map[int64][]model.SupermapCheckinimage, error) {
	var images []model.SupermapCheckinimage
	err := tx.Model(&model.SupermapCheckinimage{}).Where("status in (?)", status).Where("check_in_id IN (?)", checkinIds).Order("sort_index ASC").Find(&images).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make(map[int64][]model.SupermapCheckinimage), nil
		}
		return nil, err
	}
	var imageMap map[int64][]model.SupermapCheckinimage = make(map[int64][]model.SupermapCheckinimage)
	// 根据checkin_id分组
	for _, image := range images {
		imageMap[image.CheckInID] = append(imageMap[image.CheckInID], image)
	}
	return imageMap, nil
}

// 给指定的checkin_tags 每个的引用次数+1
func IncreaseCheckinTagReferenceCount(tx *gorm.DB, tagIds []int64) error {
	if len(tagIds) == 0 {
		return nil
	}
	err := tx.Model(&model.SupermapCheckintag{}).Where("id IN (?)", tagIds).Update("reference_count", gorm.Expr("reference_count + 1")).Error
	if err != nil {
		return err
	}
	return nil
}

type CheckinPublishRestrictResponse struct {
	IsVip               bool  `json:"is_vip"`
	MaxPublishCount     int64 `json:"max_publish_count"`
	CurrentPublishCount int64 `json:"current_publish_count"`
	CanPublish          bool  `json:"can_publish"`
}

func CheckinPublishRestrict(tx *gorm.DB, userId int64) (CheckinPublishRestrictResponse, error) {
	var isVip bool = false
	var member model.SupermapCheckinmember
	err := tx.Model(&model.SupermapCheckinmember{}).Where("user_id = ?", userId).First(&member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			isVip = false
		} else {
			return CheckinPublishRestrictResponse{}, err
		}
	} else {
		if member.ExpiredTimestamp > int32(time.Now().Unix()) {
			isVip = true
		} else {
			isVip = false
		}
	}

	// 检查用户当前发布数量
	var currentPublishCount int64 = 0
	var userStat model.SupermapUserstat
	err = tx.Model(&model.SupermapUserstat{}).Where("user_id = ?", userId).First(&userStat).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return CheckinPublishRestrictResponse{}, err
		} else {
			currentPublishCount = 0
		}
	} else {
		currentPublishCount = int64(userStat.CheckinCount)
	}

	// 从配置表读取最大发布数量 config
	var config model.Config
	maxPublishCount := int64(20)
	err = tx.Model(&model.Config{}).Where("key = ?", "checkin_publish_max_count").First(&config).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			maxPublishCount = 20
		} else {
			return CheckinPublishRestrictResponse{}, err
		}
	} else {
		maxPublishCount, err = strconv.ParseInt(config.Value, 10, 64)
		if err != nil {
			maxPublishCount = 20
		}
	}

	// 检查用户是否可以发布
	var canPublish bool = false
	if isVip {
		canPublish = true
	} else {
		if currentPublishCount < maxPublishCount {
			canPublish = true
		} else {
			canPublish = false
		}
	}
	return CheckinPublishRestrictResponse{
		IsVip:               isVip,
		MaxPublishCount:     maxPublishCount,
		CurrentPublishCount: currentPublishCount,
		CanPublish:          canPublish,
	}, nil
}
