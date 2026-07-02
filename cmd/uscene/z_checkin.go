package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db"
	"TMA/service/db/model"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const dailyUserCheckInLimit = 99

// 创建地标请求参数
type CreateCheckInRequest struct {
	CheckInPointID int64    `json:"check_in_point_id"`
	Content        string   `json:"content"`
	ImageUrls      []string `json:"image_urls"` // 图片URL列表
	Tags           []int64  `json:"tags"`       // 标签
	Private        bool     `json:"private"`    // 是否私密

}

func CreateCheckIn(ec *middleware.AppRequestContext) error {
	req := CreateCheckInRequest{}
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("[createCheckIn] 解析请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// 2. 参数验证
	if req.Content == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	ctx := ec.Request().Context()

	// 检查地标是否存在
	checkinPoint := &model.CheckInPoint{}
	err := ec.Nu.DB.Where("id = ?", req.CheckInPointID).Where("status = ?", common.SpotAuditStatusApproved).First(checkinPoint).Error
	if err != nil {
		sdlog.Errorf("[createCheckIn] 检查地标是否存在失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查用户是否可以发布
	restrict, err := db.CheckinPublishRestrict(ec.Nu.DB, ec.AuthData.User.ID)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	if !restrict.CanPublish && !restrict.IsVip {
		return webapi.Error(common.ErrCheckinUpgradeMemberToPublish).Render(ec)
	}

	slotKey, allowed, err := redis.AcquireDailyUserCheckinSlot(ctx, ec.Nu, userId, dailyUserCheckInLimit)
	if err != nil {
		sdlog.Errorf("[createCheckIn] 获取每日打卡额度失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if !allowed {
		return webapi.Error(common.ErrCheckInOverLimitDaily).Render(ec)
	}
	quotaCommitted := false
	defer func() {
		if quotaCommitted || slotKey == "" {
			return
		}
		if releaseErr := redis.ReleaseDailyUserCheckinSlot(ctx, ec.Nu, slotKey); releaseErr != nil {
			sdlog.Errorf("[createCheckIn] 释放每日打卡额度失败: %v", releaseErr)
		}
	}()

	// 开启事务
	tx := ec.Nu.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建地标
	checkin := &model.SupermapCheckin{
		CheckInPointID: req.CheckInPointID,
		UserID:         ec.AuthData.User.ID,
		Content:        req.Content,
		Status:         int32(common.CheckInStatusPending),
		Private:        req.Private,
	}
	err = tx.Create(checkin).Error
	if err != nil {
		sdlog.Errorf("[createCheckIn] 创建checkin失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 写入图片
	for i, imageUrl := range req.ImageUrls {
		image := &model.SupermapCheckinimage{
			ImageURL:  fmt.Sprintf("%s/%s", ec.Nu.Config.Aws.AwsS3CustomDomain, imageUrl),
			SortIndex: int32(i),
			Status:    int32(common.UserImageStatusPending),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CheckInID: checkin.ID,
		}
		err = tx.Create(image).Error
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("[createCheckIn] 创建图片失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	// 写入tag
	for _, tagId := range req.Tags {
		tag := &model.SupermapCheckintagrelation{
			CheckInTagID: tagId,
			CheckInID:    checkin.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err = tx.Create(tag).Error
		if err != nil {
			tx.Rollback()
			sdlog.Errorf("[createCheckIn] 创建tag失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	// increase checkin tag reference count
	_ = db.IncreaseCheckinTagReferenceCount(tx, req.Tags)

	// update user stat checkin count
	_ = service.CreateUserStat(ec.Request().Context(), ec.Nu, ec.AuthData.User.ID)
	_ = tx.Model(&model.SupermapUserstat{}).Where("checkin_count + ? >= 0", 1).Where("user_id = ?", ec.AuthData.User.ID).Update("checkin_count", gorm.Expr("checkin_count + ?", 1)).Error

	// update checkin point checkin count
	err = tx.Model(&model.CheckInPoint{}).Where("id = ?", req.CheckInPointID).Update("checkin_count", gorm.Expr("checkin_count + ?", 1)).Error
	if err != nil {
		tx.Rollback()
		sdlog.Errorf("[createCheckIn] 更新地标打卡数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	err = tx.Commit().Error
	if err != nil {
		sdlog.Errorf("[createCheckIn] 提交事务失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	quotaCommitted = true
	return webapi.OK(true).Render(ec)
}

type CheckinPointStruct struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Location    string  `json:"location"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type UserStruct struct {
	UserID    int64  `json:"user_id"`
	AvatarURL string `json:"avatar_url"`
	Nickname  string `json:"nickname"`
}

type TagsStruct struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CheckinResponse struct {
	ID             int64                        `json:"id"`
	Content        string                       `json:"content"`
	Status         int32                        `json:"status"`
	Private        bool                         `json:"private"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	CheckInPointID int64                        `json:"check_in_point_id"`
	UserID         int64                        `json:"user_id"`
	Images         []model.SupermapCheckinimage `json:"images"`
	Tags           []TagsStruct                 `json:"tags"`

	Point CheckinPointStruct `json:"point"`
	User  UserStruct         `json:"user"`

	CommentCount   int32 `json:"comment_count"`
	LikeCount      int32 `json:"like_count"`
	IsUserFollowed bool  `json:"is_user_followed"`
	IsLiked        bool  `json:"is_liked"`
}

func getCheckin(ec *middleware.AppRequestContext) error {

	checkinId := sdparse.Int64Def(ec.QueryParams().Get("checkin_id"), 0)
	if checkinId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var checkin model.SupermapCheckin
	err := ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", checkinId).First(&checkin).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if checkin.UserID != ec.AuthData.User.ID && checkin.Status != int32(common.CheckInStatusApproved) {
		return webapi.Error(common.ErrCheckInNotExists).Render(ec)
	}

	var checkinPoint CheckinPointStruct
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", checkin.CheckInPointID).Where("status = ?", common.SpotAuditStatusApproved).First(&checkinPoint).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var user UserStruct
	err = ec.Nu.DB.Model(&model.User{}).Where("user_id = ?", checkin.UserID).First(&user).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var images []model.SupermapCheckinimage
	err = ec.Nu.DB.Model(&model.SupermapCheckinimage{}).Where("check_in_id = ?", checkinId).Where("status = ?", common.UserImageStatusApproved).Find(&images).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var tags []model.SupermapCheckintagrelation
	err = ec.Nu.DB.Model(&model.SupermapCheckintagrelation{}).Where("check_in_id = ?", checkinId).Find(&tags).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var tagsResponse []TagsStruct
	// 转换tags， 先查询所有的tag
	var allTags []model.SupermapCheckintag
	err = ec.Nu.DB.Model(&model.SupermapCheckintag{}).Find(&allTags).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	tagMap := make(map[int64]model.SupermapCheckintag)
	for _, tag := range allTags {
		tagMap[tag.ID] = tag
	}
	for _, tag := range tags {
		if _, ok := tagMap[tag.CheckInTagID]; ok {
			tagsResponse = append(tagsResponse, TagsStruct{
				ID:   tag.CheckInTagID,
				Name: tagMap[tag.CheckInTagID].Name,
			})
		}
	}

	// 查询是否关注该用户，user_follow表
	var isUserFollowed bool
	if ec.AuthData.User.ID == checkin.UserID {
		isUserFollowed = true
	} else {
		err = ec.Nu.DB.Model(&model.SupermapUserfollow{}).Where("user_id = ? and followed_user_id = ?", ec.AuthData.User.ID, checkin.UserID).First(&model.SupermapUserfollow{}).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				isUserFollowed = false
			} else {
				return webapi.Error(common.ErrService).Render(ec)
			}
		} else {
			isUserFollowed = true
		}
	}

	// checkin point visit count + 1
	_ = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", checkin.CheckInPointID).Update("checkin_visit_count", gorm.Expr("checkin_visit_count + 1")).Error
	_ = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", checkinId).Update("visit_count", gorm.Expr("visit_count + 1")).Error
	// check is liked
	var isLiked bool = false
	err = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id = ? and user_id = ?", checkinId, ec.AuthData.User.ID).First(&model.SupermapCheckinlike{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			isLiked = false
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	} else {
		isLiked = true
	}

	var checkinResponse = CheckinResponse{
		ID:             checkin.ID,
		Content:        checkin.Content,
		Status:         checkin.Status,
		Private:        checkin.Private,
		CreatedAt:      checkin.CreatedAt,
		UpdatedAt:      checkin.UpdatedAt,
		CheckInPointID: checkin.CheckInPointID,
		UserID:         checkin.UserID,
		Images:         images,
		Tags:           tagsResponse,
		Point: CheckinPointStruct{
			ID:          checkin.CheckInPointID,
			Title:       checkinPoint.Title,
			Country:     checkinPoint.Country,
			City:        checkinPoint.City,
			Region:      checkinPoint.Region,
			Location:    checkinPoint.Location,
			Description: checkinPoint.Description,
			Latitude:    checkinPoint.Latitude,
			Longitude:   checkinPoint.Longitude,
		},
		User: UserStruct{
			UserID:    checkin.UserID,
			AvatarURL: user.AvatarURL,
			Nickname:  user.Nickname,
		},
		CommentCount:   checkin.CommentCount,
		LikeCount:      checkin.LikeCount,
		IsUserFollowed: isUserFollowed,
		IsLiked:        isLiked,
	}
	return webapi.OK(checkinResponse).Render(ec)
}

type CheckinTagResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func getCheckinTags(ec *middleware.AppRequestContext) error {

	var checkinTags []model.SupermapCheckintag
	err := ec.Nu.DB.Model(&model.SupermapCheckintag{}).Find(&checkinTags).Order("reference_count DESC").Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	response := make([]CheckinTagResponse, len(checkinTags))
	for i, tag := range checkinTags {
		response[i] = CheckinTagResponse{
			ID:   tag.ID,
			Name: tag.Name,
		}
	}
	return webapi.OK(response).Render(ec)
}

type UserCheckinListResponse struct {
	TotalCount int64             `json:"total_count"`
	Checkins   []CheckinResponse `json:"checkins"`
	Page       int64             `json:"page"`
	PageSize   int64             `json:"page_size"`
}

// 用户的打卡列表
func userCheckinList(ec *middleware.AppRequestContext) error {

	userId := ec.AuthData.User.ID
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	userIdStr := ec.QueryParams().Get("user_id")
	if userIdStr != "" {
		userIdInt, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return webapi.Error(common.ErrParam).Render(ec)
		}
		userId = userIdInt
	}
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	isSelf := userId == ec.AuthData.User.ID
	var total int64
	builder := ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("user_id = ?", userId)
	if !isSelf {
		builder = builder.Where("private = ?", false).Where("status = ?", common.CheckInStatusApproved)
	}
	err = builder.Count(&total).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询总数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkins []model.SupermapCheckin
	err = builder.Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&checkins).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询打卡列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkinIds []int64
	var userIds []int64
	var checkinPointIds []int64

	for _, checkin := range checkins {
		checkinIds = append(checkinIds, checkin.ID)
		userIds = append(userIds, checkin.UserID)
		checkinPointIds = append(checkinPointIds, checkin.CheckInPointID)
	}
	var checkinPoints []model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id IN (?)", checkinPointIds).Find(&checkinPoints).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询地标列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkinPointStructMap map[int64]CheckinPointStruct = make(map[int64]CheckinPointStruct)
	for _, checkinPoint := range checkinPoints {
		checkinPointStructMap[checkinPoint.ID] = CheckinPointStruct{
			ID:          checkinPoint.ID,
			Title:       checkinPoint.Title,
			Country:     checkinPoint.Country,
			City:        checkinPoint.City,
			Region:      checkinPoint.Region,
			Location:    checkinPoint.Location,
			Description: checkinPoint.Description,
			Latitude:    checkinPoint.Latitude,
			Longitude:   checkinPoint.Longitude,
		}
	}

	// users
	var users []model.User
	err = ec.Nu.DB.Model(&model.User{}).Where("user_id IN (?)", userIds).Find(&users).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询用户列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var userStructMap map[int64]UserStruct = make(map[int64]UserStruct)
	for _, user := range users {
		userStructMap[user.UserID] = UserStruct{
			UserID:    user.UserID,
			AvatarURL: user.AvatarURL,
			Nickname:  user.Nickname,
		}
	}

	var images []model.SupermapCheckinimage
	err = ec.Nu.DB.Model(&model.SupermapCheckinimage{}).Where("check_in_id IN (?)", checkinIds).Find(&images).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询图片列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var imageMap map[int64][]model.SupermapCheckinimage = make(map[int64][]model.SupermapCheckinimage)
	for _, image := range images {
		imageMap[image.CheckInID] = append(imageMap[image.CheckInID], image)
	}

	// tags
	var tagConfigs []model.SupermapCheckintag
	err = ec.Nu.DB.Model(&model.SupermapCheckintag{}).Find(&tagConfigs).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询tag配置列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var tagCongigMap map[int64]model.SupermapCheckintag = make(map[int64]model.SupermapCheckintag)
	for _, tagConfig := range tagConfigs {
		tagCongigMap[tagConfig.ID] = tagConfig
	}

	var tags []model.SupermapCheckintagrelation
	err = ec.Nu.DB.Model(&model.SupermapCheckintagrelation{}).Where("check_in_id IN (?)", checkinIds).Find(&tags).Error
	if err != nil {
		sdlog.Errorf("[userCheckinList] 查询tag关系列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var tagMap map[int64][]TagsStruct = make(map[int64][]TagsStruct)
	for _, tag := range tags {
		if _, ok := tagCongigMap[tag.CheckInTagID]; ok {
			tagMap[tag.CheckInID] = append(tagMap[tag.CheckInID], TagsStruct{
				ID:   tag.CheckInTagID,
				Name: tagCongigMap[tag.CheckInTagID].Name,
			})
		}
	}

	var likes []model.SupermapCheckinlike
	_ = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id IN (?) and user_id = ?", checkinIds, ec.AuthData.User.ID).Find(&likes).Error
	var likeMap map[int64]bool = make(map[int64]bool)
	for _, like := range likes {
		likeMap[like.CheckInID] = true
	}

	var checkinPointsResponse UserCheckinListResponse
	checkinPointsResponse.TotalCount = total
	checkinPointsResponse.Page = int64(pageInt)
	checkinPointsResponse.PageSize = int64(pageSizeInt)

	for _, checkin := range checkins {
		var like bool = false
		if _, ok := likeMap[checkin.ID]; ok {
			like = true
		}
		checkinItem := CheckinResponse{
			ID:             checkin.ID,
			Content:        checkin.Content,
			Status:         checkin.Status,
			Private:        checkin.Private,
			CreatedAt:      checkin.CreatedAt,
			UpdatedAt:      checkin.UpdatedAt,
			CheckInPointID: checkin.CheckInPointID,
			UserID:         checkin.UserID,
			Images:         images,
			CommentCount:   checkin.CommentCount,
			LikeCount:      checkin.LikeCount,
			IsUserFollowed: false,
			IsLiked:        like,
		}

		if _, ok := userStructMap[checkin.UserID]; ok {
			checkinItem.User = userStructMap[checkin.UserID]
		}

		if _, ok := checkinPointStructMap[checkin.CheckInPointID]; ok {
			checkinItem.Point = checkinPointStructMap[checkin.CheckInPointID]
		}

		if _, ok := imageMap[checkin.ID]; ok {
			checkinItem.Images = imageMap[checkin.ID]
		}

		if _, ok := tagMap[checkin.ID]; ok {
			checkinItem.Tags = tagMap[checkin.ID]
		}

		checkinPointsResponse.Checkins = append(checkinPointsResponse.Checkins, checkinItem)
	}

	return webapi.OK(checkinPointsResponse).Render(ec)
}

type AddCheckinCommentRequest struct {
	CheckinID int64  `json:"checkin_id"`
	Content   string `json:"content"`
	City      string `json:"city"`
}

// 给打卡添加评论
func addCheckinComment(ec *middleware.AppRequestContext) error {

	reqParam := AddCheckinCommentRequest{}
	if err := ec.Bind(&reqParam); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	if len(reqParam.Content) == 0 {
		return webapi.OK(common.ErrParam).Render(ec)
	}
	// check checkin id is exists
	var checkin model.SupermapCheckin
	err := ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	comment := &model.SupermapCheckincomment{
		CheckInID:     reqParam.CheckinID,
		UserID:        userId,
		ParentID:      0,
		ReplyToID:     int32(0),
		ReplyToUserID: 0,
		LikeCount:     int32(0),
		Status:        int32(common.CheckInCommentStatusNormal),
		Content:       reqParam.Content,
		City:          reqParam.City,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = ec.Nu.DB.Create(comment).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 更新checkin point comment count + 1
	_ = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", checkin.CheckInPointID).Update("checkin_comment_count", gorm.Expr("checkin_comment_count + 1")).Error
	// 更新checkin comment count + 1
	_ = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).Update("comment_count", gorm.Expr("comment_count + 1")).Error

	// 互动消息写入supermap_checkinmessage
	// 如果点赞，插入新消息，如果取消点赞，删除原消息
	if checkin.UserID != ec.AuthData.User.ID {
		_ = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Create(&model.SupermapCheckinmessage{
			CheckInID:   reqParam.CheckinID,
			UserID:      checkin.UserID,
			MessageType: common.CheckInMessageTypeComment,
			OtherUserID: ec.AuthData.User.ID,
			Content:     "评论: " + reqParam.Content,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}).Error
	}
	return webapi.OK(true).Render(ec)
}

type DeleteCheckinCommentRequest struct {
	CheckinID int64 `json:"checkin_id"`
	CommentID int64 `json:"comment_id"`
}

// 删除打卡评论
func deleteCheckinComment(ec *middleware.AppRequestContext) error {

	reqParam := DeleteCheckinCommentRequest{}
	if err := ec.Bind(&reqParam); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	// check checkin id is exists
	var checkin model.SupermapCheckin
	err := ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// check checkin comment id is exists
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("id = ? and user_id = ? and check_in_id = ?", reqParam.CommentID, userId, reqParam.CheckinID).First(&model.SupermapCheckincomment{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInCommentNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// delete checkin comment
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("id = ? and user_id = ? and check_in_id = ?", reqParam.CommentID, userId, reqParam.CheckinID).Delete(&model.SupermapCheckincomment{}).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新checkin point comment count - 1,更新后数量不能小于0
	_ = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ? and checkin_comment_count > 0", checkin.CheckInPointID).Update("checkin_comment_count", gorm.Expr("checkin_comment_count - 1")).Error

	// 更新checkin comment count - 1,更新后数量不能小于0
	_ = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ? and comment_count > 0", reqParam.CheckinID).Update("comment_count", gorm.Expr("comment_count - 1")).Error

	return webapi.OK(true).Render(ec)
}

type ReplyCheckinCommentRequest struct {
	CheckinID int64  `json:"checkin_id"`
	CommentID int64  `json:"comment_id"`
	Content   string `json:"content"`
	City      string `json:"city"`
}

// 回复打卡评论
func replyCheckinComment(ec *middleware.AppRequestContext) error {

	reqParam := ReplyCheckinCommentRequest{}
	if err := ec.Bind(&reqParam); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if len(reqParam.Content) == 0 {
		return webapi.OK(common.ErrParam).Render(ec)
	}

	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	// check checkin id is exists
	var checkin model.SupermapCheckin
	err := ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var comment model.SupermapCheckincomment
	// check comment id is exists
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("id = ?", reqParam.CommentID).First(&comment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInCommentNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	var parentId int32
	if comment.ParentID != 0 {
		parentId = comment.ParentID
	} else {
		parentId = int32(reqParam.CommentID)
	}

	newComment := &model.SupermapCheckincomment{
		CheckInID:     reqParam.CheckinID,
		UserID:        userId,
		ParentID:      parentId,
		ReplyToID:     int32(reqParam.CommentID),
		ReplyToUserID: comment.UserID,
		LikeCount:     int32(0),
		Status:        int32(common.CheckInCommentStatusNormal),
		Content:       reqParam.Content,
		City:          reqParam.City,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = ec.Nu.DB.Create(newComment).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新checkin point comment count + 1
	_ = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", reqParam.CheckinID).Update("checkin_comment_count", gorm.Expr("checkin_comment_count + 1")).Error

	// 更新checkin comment count + 1
	_ = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).Update("comment_count", gorm.Expr("comment_count + 1")).Error
	// 互动消息写入supermap_checkinmessage
	// 如果回复，插入新消息
	if checkin.UserID != ec.AuthData.User.ID {
		_ = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Create(&model.SupermapCheckinmessage{
			CheckInID:   reqParam.CheckinID,
			UserID:      checkin.UserID,
			MessageType: common.CheckInMessageTypeComment,
			OtherUserID: ec.AuthData.User.ID,
			Content:     "回复: " + reqParam.Content,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}).Error
	}
	return webapi.OK(true).Render(ec)
}

type SpotLatestCheckinResponse struct {
	ID             int64                        `json:"id"`
	Content        string                       `json:"content"`
	Status         int32                        `json:"status"`
	Private        bool                         `json:"private"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	CheckInPointID int64                        `json:"check_in_point_id"`
	UserID         int64                        `json:"user_id"`
	Images         []model.SupermapCheckinimage `json:"images"`
	Tags           []db.TagsStruct              `json:"tags"`

	User           UserStruct `json:"user"`
	CommentCount   int32      `json:"comment_count"`
	LikeCount      int32      `json:"like_count"`
	IsUserFollowed bool       `json:"is_user_followed"`
	IsLiked        bool       `json:"is_liked"`

	Point CheckinPointStruct `json:"point"`
}

type SpotLatestCheckinListResponse struct {
	TotalCount int64                       `json:"total_count"`
	Checkins   []SpotLatestCheckinResponse `json:"checkins"`
	Page       int64                       `json:"page"`
	PageSize   int64                       `json:"page_size"`
}

// 地标下的打卡列表 , 分页, get 请求
func getSpotLatestCheckinList(ec *middleware.AppRequestContext) error {

	spotId := ec.QueryParams().Get("spot_id")
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")

	if spotId == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if page == "" {
		page = "1"
	}

	if pageSize == "" {
		pageSize = "10"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var total int64
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("check_in_point_id = ?", spotId).Count(&total).Error
	if err != nil {
		sdlog.Errorf("[getSpotLatestCheckinList] 查询总数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var spot model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", spotId).First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrSpotNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkins []model.SupermapCheckin
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("check_in_point_id = ?", spotId).Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&checkins).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var userIds []int64
	var checkinIds []int64

	for _, checkin := range checkins {
		userIds = append(userIds, checkin.UserID)
		checkinIds = append(checkinIds, checkin.ID)
	}

	// users
	var users []model.User
	err = ec.Nu.DB.Model(&model.User{}).Where("user_id IN (?)", userIds).Find(&users).Error
	if err != nil {
		sdlog.Errorf("[getSpotLatestCheckinList] 查询用户列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var userStructMap map[int64]UserStruct = make(map[int64]UserStruct)
	for _, user := range users {
		userStructMap[user.UserID] = UserStruct{
			UserID:    user.UserID,
			AvatarURL: user.AvatarURL,
			Nickname:  user.Nickname,
		}
	}

	tagMap, err := db.GetCheckinTagsMap(ec.Nu.DB, checkinIds)
	if err != nil {
		sdlog.Errorf("[getSpotLatestCheckinList] 查询tag列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	imageMap, err := db.GetCheckinImagesMap(ec.Nu.DB, checkinIds, []int32{int32(common.UserImageStatusApproved)})
	if err != nil {
		sdlog.Errorf("[getSpotLatestCheckinList] 查询图片列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// like
	var likes []model.SupermapCheckinlike
	_ = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id IN (?) and user_id = ?", checkinIds, ec.AuthData.User.ID).Find(&likes).Error
	var likeMap map[int64]bool = make(map[int64]bool)
	for _, like := range likes {
		likeMap[like.CheckInID] = true
	}

	var spotLatestCheckinListResponse SpotLatestCheckinListResponse
	spotLatestCheckinListResponse.TotalCount = total
	spotLatestCheckinListResponse.Page = int64(pageInt)
	spotLatestCheckinListResponse.PageSize = int64(pageSizeInt)

	for _, checkin := range checkins {
		var checkinTags []db.TagsStruct
		if _, ok := tagMap[checkin.ID]; ok {
			checkinTags = tagMap[checkin.ID]
		}
		var checkinImages []model.SupermapCheckinimage = make([]model.SupermapCheckinimage, 0)
		if _, ok := imageMap[checkin.ID]; ok {
			checkinImages = imageMap[checkin.ID]
		}
		var like bool = false
		if _, ok := likeMap[checkin.ID]; ok {
			like = true
		}
		spotLatestCheckinListResponse.Checkins = append(spotLatestCheckinListResponse.Checkins, SpotLatestCheckinResponse{
			ID:             checkin.ID,
			Content:        checkin.Content,
			Status:         checkin.Status,
			Private:        checkin.Private,
			CreatedAt:      checkin.CreatedAt,
			UpdatedAt:      checkin.UpdatedAt,
			CheckInPointID: checkin.CheckInPointID,
			UserID:         checkin.UserID,
			Images:         checkinImages,
			Tags:           checkinTags,
			User:           userStructMap[checkin.UserID],
			CommentCount:   checkin.CommentCount,
			LikeCount:      checkin.LikeCount,
			IsUserFollowed: false,
			IsLiked:        like,
			Point: CheckinPointStruct{
				ID:          spot.ID,
				Title:       spot.Title,
				Country:     spot.Country,
				City:        spot.City,
				Region:      spot.Region,
				Location:    spot.Location,
				Description: spot.Description,
				Latitude:    spot.Latitude,
				Longitude:   spot.Longitude,
			},
		})
	}

	return webapi.OK(spotLatestCheckinListResponse).Render(ec)
}

func getSpotRecommendedCheckinList(ec *middleware.AppRequestContext) error {

	spotId := ec.QueryParams().Get("spot_id")
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")

	if spotId == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if page == "" {
		page = "1"
	}

	if pageSize == "" {
		pageSize = "10"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var spot model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ?", spotId).First(&spot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrSpotNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var total int64
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("check_in_point_id = ?", spotId).Count(&total).Error
	if err != nil {
		sdlog.Errorf("[getSpotRecommendedCheckinList] 查询总数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkins []model.SupermapCheckin
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("check_in_point_id = ?", spotId).Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&checkins).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var userIds []int64
	var checkinIds []int64

	for _, checkin := range checkins {
		userIds = append(userIds, checkin.UserID)
		checkinIds = append(checkinIds, checkin.ID)
	}

	var users []model.User
	err = ec.Nu.DB.Model(&model.User{}).Where("user_id IN (?)", userIds).Find(&users).Error
	if err != nil {
		sdlog.Errorf("[getSpotRecommendedCheckinList] 查询用户列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var userStructMap map[int64]UserStruct = make(map[int64]UserStruct)
	for _, user := range users {
		userStructMap[user.UserID] = UserStruct{
			UserID:    user.UserID,
			AvatarURL: user.AvatarURL,
			Nickname:  user.Nickname,
		}
	}

	tagMap, err := db.GetCheckinTagsMap(ec.Nu.DB, checkinIds)
	if err != nil {
		sdlog.Errorf("[getSpotRecommendedCheckinList] 查询tag列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	imageMap, err := db.GetCheckinImagesMap(ec.Nu.DB, checkinIds, []int32{int32(common.UserImageStatusApproved)})
	if err != nil {
		sdlog.Errorf("[getSpotRecommendedCheckinList] 查询图片列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// like
	var likes []model.SupermapCheckinlike
	_ = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id IN (?) and user_id = ?", checkinIds, ec.AuthData.User.ID).Find(&likes).Error
	var likeMap map[int64]bool = make(map[int64]bool)
	for _, like := range likes {
		likeMap[like.CheckInID] = true
	}

	var spotRecommendedCheckinListResponse SpotRecommendedCheckinListResponse
	spotRecommendedCheckinListResponse.TotalCount = total
	spotRecommendedCheckinListResponse.Page = int64(pageInt)
	spotRecommendedCheckinListResponse.PageSize = int64(pageSizeInt)

	for _, checkin := range checkins {
		var checkinTags []db.TagsStruct
		if _, ok := tagMap[checkin.ID]; ok {
			checkinTags = tagMap[checkin.ID]
		}
		var checkinImages []model.SupermapCheckinimage = make([]model.SupermapCheckinimage, 0)
		if _, ok := imageMap[checkin.ID]; ok {
			checkinImages = imageMap[checkin.ID]
		}
		var like bool = false
		if _, ok := likeMap[checkin.ID]; ok {
			like = true
		}
		spotRecommendedCheckinListResponse.Checkins = append(spotRecommendedCheckinListResponse.Checkins, &SpotLatestCheckinResponse{
			ID:             checkin.ID,
			Content:        checkin.Content,
			Status:         checkin.Status,
			Private:        checkin.Private,
			CreatedAt:      checkin.CreatedAt,
			UpdatedAt:      checkin.UpdatedAt,
			CheckInPointID: checkin.CheckInPointID,
			UserID:         checkin.UserID,
			Images:         checkinImages,
			Tags:           checkinTags,
			User:           userStructMap[checkin.UserID],
			CommentCount:   checkin.CommentCount,
			LikeCount:      checkin.LikeCount,
			IsUserFollowed: false,
			IsLiked:        like,
			Point: CheckinPointStruct{
				ID:          spot.ID,
				Title:       spot.Title,
				Country:     spot.Country,
				City:        spot.City,
				Region:      spot.Region,
				Location:    spot.Location,
				Description: spot.Description,
				Latitude:    spot.Latitude,
				Longitude:   spot.Longitude,
			},
		})
	}

	return webapi.OK(spotRecommendedCheckinListResponse).Render(ec)
}

type SpotRecommendedCheckinListResponse struct {
	TotalCount int64                        `json:"total_count"`
	Checkins   []*SpotLatestCheckinResponse `json:"checkins"`
	Page       int64                        `json:"page"`
	PageSize   int64                        `json:"page_size"`
}

type CheckinCommentListResponse struct {
	TotalCommentCount int64                `json:"total_comment_count"`
	TotalPage         int64                `json:"total_page"`
	Comments          []CheckinCommentItem `json:"comments"`
	Page              int64                `json:"page"`
	PageSize          int64                `json:"page_size"`
}

type CheckinCommentUser struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type CheckinCommetWithUser struct {
	Comment     *model.SupermapCheckincomment `json:"comment"`
	User        CheckinCommentUser            `json:"user"`
	ReplyToUser CheckinCommentUser            `json:"reply_to_user"`
	IsLiked     bool                          `json:"is_liked"`
}

type CheckinCommentItem struct {
	Comment CheckinCommetWithUser    `json:"comment"`
	Replies []*CheckinCommetWithUser `json:"replies"`
}

// page page_size, 先查找一级评论，然后二级评论添加到一级评论的子评论列表中
func getCheckinCommentList(ec *middleware.AppRequestContext) error {

	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")

	if page == "" {
		page = "1"
	}

	if pageSize == "" {
		pageSize = "10"
	}

	pageInt, err := sdparse.Int(page)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeInt, err := sdparse.Int(pageSize)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	checkinId := ec.QueryParams().Get("checkin_id")
	if checkinId == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	checkinIdInt, err := sdparse.Int64(checkinId)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var total int64
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("check_in_id = ?", checkinIdInt).Count(&total).Error
	if err != nil {
		sdlog.Errorf("[getCheckinCommentList] 查询总数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var totalParentCommentCount int64
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("check_in_id = ? and parent_id = ?", checkinIdInt, 0).Count(&totalParentCommentCount).Error
	if err != nil {
		sdlog.Errorf("[getCheckinCommentList] 查询总数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var totalPage int64 = totalParentCommentCount / int64(pageSizeInt)
	if totalParentCommentCount%int64(pageSizeInt) != 0 {
		totalPage++
	}

	var comments []*model.SupermapCheckincomment = make([]*model.SupermapCheckincomment, 0)
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("check_in_id = ? and parent_id = ?", checkinIdInt, 0).Order("created_at DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&comments).Error
	if err != nil {
		sdlog.Errorf("[getCheckinCommentList] 查询评论列表失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(CheckinCommentListResponse{
				TotalCommentCount: 0,
				TotalPage:         0,
				Comments:          []CheckinCommentItem{},
				Page:              int64(pageInt),
				PageSize:          int64(pageSizeInt),
			}).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 根据comment_id 查询回复列表
	var allReplies []*model.SupermapCheckincomment = make([]*model.SupermapCheckincomment, 0)
	var parentIds []int64 = make([]int64, 0)
	for _, comment := range comments {
		parentIds = append(parentIds, comment.ID)
	}
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("parent_id IN (?)", parentIds).Find(&allReplies).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("[getCheckinCommentList] 查询回复列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var userIds []int64 = make([]int64, 0)
	userIds = append(userIds, ec.AuthData.User.ID)
	for _, reply := range allReplies {
		userIds = append(userIds, reply.UserID)
		userIds = append(userIds, reply.ReplyToUserID)
	}

	var replyMap map[int64][]*model.SupermapCheckincomment = make(map[int64][]*model.SupermapCheckincomment)
	for _, reply := range allReplies {
		replyMap[int64(reply.ParentID)] = append(replyMap[int64(reply.ParentID)], reply)
	}
	var users []model.User = make([]model.User, 0)
	err = ec.Nu.DB.Model(&model.User{}).Where("user_id IN (?)", userIds).Find(&users).Error
	if err != nil {
		sdlog.Errorf("[getCheckinCommentList] 查询用户列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var userStructMap map[int64]CheckinCommentUser = make(map[int64]CheckinCommentUser)
	for _, user := range users {
		userStructMap[user.UserID] = CheckinCommentUser{
			ID:        user.UserID,
			Nickname:  user.Nickname,
			AvatarURL: user.AvatarURL,
		}
	}

	var commentIds []int64 = make([]int64, 0)
	for _, comment := range comments {
		commentIds = append(commentIds, comment.ID)
	}
	for _, reply := range allReplies {
		commentIds = append(commentIds, reply.ID)
	}
	var likes []model.SupermapCheckincommentlike = make([]model.SupermapCheckincommentlike, 0)
	err = ec.Nu.DB.Model(&model.SupermapCheckincommentlike{}).Where("comment_id IN (?)", commentIds).Find(&likes).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("[getCheckinCommentList] 查询点赞列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var likeMap map[int64]bool = make(map[int64]bool)
	for _, like := range likes {
		likeMap[like.CommentID] = true
	}
	var commentsItem []CheckinCommentItem = make([]CheckinCommentItem, 0)
	for _, comment := range comments {
		// replies
		var repliesItem []*model.SupermapCheckincomment = make([]*model.SupermapCheckincomment, 0)
		if replies, ok := replyMap[comment.ID]; ok {
			repliesItem = replies
		}

		var replyItems []*CheckinCommetWithUser = make([]*CheckinCommetWithUser, 0)
		for _, reply := range repliesItem {
			var replyToUser CheckinCommentUser
			if reply.ReplyToUserID != 0 {
				replyToUser = userStructMap[reply.ReplyToUserID]
			}
			var user CheckinCommentUser
			if reply.UserID != 0 {
				user = userStructMap[reply.UserID]
			}
			var isReplyLiked bool = false
			if like, ok := likeMap[reply.ID]; ok {
				isReplyLiked = like
			}
			item := CheckinCommetWithUser{
				Comment:     reply,
				User:        user,
				ReplyToUser: replyToUser,
				IsLiked:     isReplyLiked,
			}
			replyItems = append(replyItems, &item)
		}

		var commentUser CheckinCommentUser
		if user, ok := userStructMap[comment.UserID]; ok {
			commentUser = user
		}
		var isCommentLiked bool = false
		if like, ok := likeMap[comment.ID]; ok {
			isCommentLiked = like
		}
		commentWithUser := CheckinCommetWithUser{
			Comment:     comment,
			User:        commentUser,
			ReplyToUser: CheckinCommentUser{},
			IsLiked:     isCommentLiked,
		}
		commentsItem = append(commentsItem, CheckinCommentItem{
			Comment: commentWithUser,
			Replies: replyItems,
		})

	}

	return webapi.OK(CheckinCommentListResponse{
		TotalCommentCount: total,
		TotalPage:         totalPage,
		Comments:          commentsItem,
		Page:              int64(pageInt),
		PageSize:          int64(pageSizeInt),
	}).Render(ec)
}

// 对打卡点赞，已经点赞的取消点赞，返回是否已经点赞， 点赞数据存储在表supermap_checkin_like，如果已经点赞，则删除点赞数据，如果未点赞，则插入点赞数据
// POST /checkin/like
type CheckinLikeRequest struct {
	CheckinID int64 `json:"checkin_id"`
}

func checkinLike(ec *middleware.AppRequestContext) error {

	var reqParam CheckinLikeRequest
	err := ec.Bind(&reqParam)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var checkin model.SupermapCheckin
	// check checkin id exists
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var isLiked bool
	err = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id = ? and user_id = ?", reqParam.CheckinID, ec.AuthData.User.ID).First(&model.SupermapCheckinlike{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			isLiked = false
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
		isLiked = false
	} else {
		isLiked = true
	}

	var changeCount int64 = 0
	if isLiked {
		err = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id = ? and user_id = ?", reqParam.CheckinID, ec.AuthData.User.ID).Delete(&model.SupermapCheckinlike{}).Error
		//
		changeCount = -1
	} else {
		err = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Create(&model.SupermapCheckinlike{
			CheckInID: reqParam.CheckinID,
			UserID:    ec.AuthData.User.ID,
		}).Error
		changeCount = 1
	}

	// update checkin like count
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).Update("like_count", gorm.Expr("like_count + ?", changeCount)).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	err = service.CreateUserStat(ec.Request().Context(), ec.Nu, ec.AuthData.User.ID)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	isLiked = !isLiked

	// update user stat like count
	_ = ec.Nu.DB.Model(&model.SupermapUserstat{}).Where("likes_count + ? >= 0", changeCount).Where("user_id = ?", ec.AuthData.User.ID).Update("likes_count", gorm.Expr("likes_count + ?", changeCount)).Error

	// 互动消息写入supermap_checkinmessage
	// 如果点赞，插入新消息，如果取消点赞，删除原消息
	if checkin.UserID != ec.AuthData.User.ID {
		if isLiked {
			_ = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Create(&model.SupermapCheckinmessage{
				CheckInID:   reqParam.CheckinID,
				UserID:      checkin.UserID,
				MessageType: common.CheckInMessageTypeLike,
				OtherUserID: ec.AuthData.User.ID,
				Content:     "赞了你",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}).Error
		} else {
			_ = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Where("check_in_id = ? and user_id = ?", reqParam.CheckinID, ec.AuthData.User.ID).Delete(&model.SupermapCheckinmessage{}).Error
		}
	}

	return webapi.OK(map[string]bool{
		"is_liked": isLiked,
	}).Render(ec)
}

type CheckinWithImage struct {
	model.SupermapCheckin
	Images []model.SupermapCheckinimage `json:"images"`
}

type CheckinSelected struct {
	model.SupermapCheckin `json:"checkin"`
	model.CheckInPoint    `json:"point"`
	model.User            `json:"user"`
	Distance              float64 `json:"distance"`
	HotValue              int64   `json:"hot_value"`
}

type CheckinSelectedWithImage struct {
	CheckinWithImage   `json:"checkin"`
	model.CheckInPoint `json:"point"`
	model.User         `json:"user"`
	Distance           float64 `json:"distance"`
	HotValue           int64   `json:"hot_value"`
	IsLiked            bool    `json:"is_liked"`
}

type CheckinSelectedListResponse struct {
	TotalCount int64                       `json:"total_count"`
	Checkins   []*CheckinSelectedWithImage `json:"checkins"`
	Page       int64                       `json:"page"`
	PageSize   int64                       `json:"page_size"`
}

// 1 根据用户传入经纬度，距离100000米范围内的打卡点, 需要连表查询,距离用mysql的距离函数计算
// 2 根据话题筛选，需要传入话题id，不传则是推荐列表
// topic_id 0 表示推荐列表，-1 表示按距离排序， 其他值则筛选话题
// page page_size
func getCheckinList(ec *middleware.AppRequestContext) error {
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}
	pageInt, err := sdparse.Int(page)
	if err != nil {
		sdlog.Errorf("[getCheckinList] 解析页码失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	pageSizeInt, err := sdparse.Int(pageSize)
	if err != nil {
		sdlog.Errorf("[getCheckinList] 解析页大小失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	latitude := ec.QueryParams().Get("latitude")
	longitude := ec.QueryParams().Get("longitude")
	if latitude == "" || longitude == "" {
		sdlog.Errorf("[getCheckinList] 解析经纬度失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	latitudeFloat, err := sdparse.Float64(latitude)
	if err != nil {
		sdlog.Errorf("[getCheckinList] 解析经纬度失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	longitudeFloat, err := sdparse.Float64(longitude)
	if err != nil {
		sdlog.Errorf("[getCheckinList] 解析经纬度失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	topicId := ec.QueryParams().Get("topic_id")
	if topicId == "" {
		topicId = "0"
	}
	topicIdInt, err := sdparse.Int64(topicId)
	if err != nil {
		sdlog.Errorf("[getCheckinList] 解析话题ID失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var checkinList []*CheckinSelected = make([]*CheckinSelected, 0)

	query := ec.Nu.DB.Model(&CheckinSelected{}).Table("supermap_checkin").Select("supermap_checkin.*, check_in_point.*, user.*, ST_Distance_Sphere(point(check_in_point.longitude, check_in_point.latitude), point(?, ?)) AS distance, supermap_checkin.comment_count * 3 + supermap_checkin.like_count * 2 + supermap_checkin.visit_count * 1 as hot_value ", longitudeFloat, latitudeFloat).
		Joins("LEFT JOIN check_in_point ON check_in_point.id = supermap_checkin.check_in_point_id").
		Joins("LEFT JOIN user ON supermap_checkin.user_id = user.user_id").
		Where("check_in_point.status = 1 and check_in_point.is_active = 1 and check_in_point.audit_status = 1").
		Where("supermap_checkin.status = ? and supermap_checkin.private = ?", common.CheckInStatusApproved, false).
		Where("ST_Distance_Sphere(point(check_in_point.longitude, check_in_point.latitude), point(?, ?)) < ?", longitudeFloat, latitudeFloat, 100000)
	if topicIdInt > 0 {
		query = query.Where(`
		EXISTS (
			SELECT 1 
			FROM supermap_checkintagrelation r 
			WHERE r.check_in_id = supermap_checkin.id 
			  AND r.check_in_tag_id = ?
		)
	`, topicIdInt)
	}
	if topicIdInt == -1 {
		query = query.Order("distance ASC")
	} else {
		query = query.Order("hot_value DESC")
	}
	var total int64
	err = query.Count(&total).Error
	if err != nil {
		sdlog.Errorf("[getCheckinList] 查询打卡点列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	err = query.Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&checkinList).Error
	if err != nil {
		sdlog.Errorf("[getCheckinList] 查询打卡点列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 查询打卡是否点赞
	var checkinIds []int64 = make([]int64, 0)
	for _, checkin := range checkinList {
		checkinIds = append(checkinIds, checkin.SupermapCheckin.ID)
	}
	var checkinLikeList []model.SupermapCheckinlike = make([]model.SupermapCheckinlike, 0)
	err = ec.Nu.DB.Model(&model.SupermapCheckinlike{}).Where("check_in_id IN (?) and user_id = ?", checkinIds, ec.AuthData.User.ID).Find(&checkinLikeList).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		sdlog.Errorf("[getCheckinList] 查询打卡是否点赞失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	var checkinLikeMap map[int64]bool = make(map[int64]bool)
	for _, checkinLike := range checkinLikeList {
		checkinLikeMap[checkinLike.CheckInID] = true
	}
	// checkimagamap
	imagesMap, err := db.GetCheckinImagesMap(ec.Nu.DB, checkinIds, []int32{common.UserImageStatusApproved})
	if err != nil {
		sdlog.Errorf("[getCheckinList] 查询图片失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkinSelectedList []*CheckinSelectedWithImage = make([]*CheckinSelectedWithImage, 0)
	for _, checkin := range checkinList {
		var images []model.SupermapCheckinimage = make([]model.SupermapCheckinimage, 0)
		if _images, ok := imagesMap[checkin.SupermapCheckin.ID]; ok {
			images = _images
		}
		var checkinSelectedWithImage CheckinSelectedWithImage = CheckinSelectedWithImage{
			CheckinWithImage: CheckinWithImage{
				SupermapCheckin: checkin.SupermapCheckin,
				Images:          images,
			},
			CheckInPoint: checkin.CheckInPoint,
			User:         checkin.User,
			Distance:     checkin.Distance,
			HotValue:     checkin.HotValue,
			IsLiked:      checkinLikeMap[checkin.SupermapCheckin.ID],
		}
		checkinSelectedList = append(checkinSelectedList, &checkinSelectedWithImage)
	}

	return webapi.OK(CheckinSelectedListResponse{
		TotalCount: total,
		Checkins:   checkinSelectedList,
		Page:       int64(pageInt),
		PageSize:   int64(pageSizeInt),
	}).Render(ec)
}

// 先从supermap_checkinmessagereadprogress 读取上次读取的id,再去supermap_checkinmessage拉取未读数量
// GET /checkin/message/unread_count
func checkinMessageUnreadCount(ec *middleware.AppRequestContext) error {
	var lastReadID int64 = 0
	err := ec.Nu.DB.Model(&model.SupermapCheckinmessagereadprogress{}).Where("user_id = ?", ec.AuthData.User.ID).Select("last_read_id").First(&lastReadID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			lastReadID = 0
		} else {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	var unreadCount int64 = 0
	err = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Where("user_id = ? and id > ?", ec.AuthData.User.ID, lastReadID).Count(&unreadCount).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(map[string]int64{
		"unread_count": unreadCount,
	}).Render(ec)
}

const (
	MessageTypeCheckinLike        = "checkin_like"
	MessageTypeCheckinComment     = "checkin_comment"
	MessageTypeCheckinCommentLike = "checkin_comment_like"
)

type CheckInWithImage struct {
	model.SupermapCheckin
	Images []model.SupermapCheckinimage `json:"images"`
}

type MessageItem struct {
	User        *model.User       `json:"user"`
	CheckIn     *CheckInWithImage `json:"checkin"`
	Content     string            `json:"content"`
	Timestamp   int64             `json:"timestamp"`
	MessageType string            `json:"message_type"`
}

type CheckInMessage struct {
	model.SupermapCheckinmessage `json:"message"`
	model.User                   `json:"user"`
	model.SupermapCheckin        `json:"checkin"`
	model.SupermapCheckincomment `json:"comment"`
}

func checkinMessageList(ec *middleware.AppRequestContext) error {
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}
	pageInt, err := sdparse.Int(page)
	if err != nil {
		sdlog.Errorf("[checkinMessageList] 解析页码失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	pageSizeInt, err := sdparse.Int(pageSize)
	if err != nil {
		sdlog.Errorf("[checkinMessageList] 解析页大小失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	var total int64 = 0
	err = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Where("user_id = ?", ec.AuthData.User.ID).Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	//连表查询
	var messages []CheckInMessage = make([]CheckInMessage, 0)
	err = ec.Nu.DB.Model(&CheckInMessage{}).Table("supermap_checkinmessage").Select("supermap_checkinmessage.* , user.* , supermap_checkin.*, supermap_checkincomment.*").
		Joins("LEFT JOIN user ON supermap_checkinmessage.other_user_id = user.user_id").
		Joins("LEFT JOIN supermap_checkin ON supermap_checkinmessage.check_in_id = supermap_checkin.id").
		Joins("LEFT JOIN supermap_checkincomment ON supermap_checkinmessage.comment_id = supermap_checkincomment.id").
		Where("supermap_checkinmessage.user_id = ?", ec.AuthData.User.ID).Order("supermap_checkinmessage.id DESC").Offset(int((pageInt - 1) * pageSizeInt)).Limit(int(pageSizeInt)).Find(&messages).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var checkinIds []int64 = make([]int64, 0)
	for _, message := range messages {
		checkinIds = append(checkinIds, message.SupermapCheckin.ID)
	}
	// 查询图片
	imageMap, err := db.GetCheckinImagesMap(ec.Nu.DB, checkinIds, []int32{common.UserImageStatusApproved})
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	var messageItems []MessageItem = make([]MessageItem, 0)
	for _, message := range messages {
		var _images []model.SupermapCheckinimage
		if images, ok := imageMap[message.SupermapCheckin.ID]; ok {
			_images = images
		}
		var messageType string
		var content string
		sdlog.Infof("[checkinMessageList] message.SupermapCheckinmessage.MessageType: %+v", message.SupermapCheckinmessage.MessageType)
		if message.SupermapCheckinmessage.MessageType == int32(common.CheckInMessageTypeLike) {
			messageType = MessageTypeCheckinLike
			content = message.SupermapCheckinmessage.Content
		} else if message.SupermapCheckinmessage.MessageType == int32(common.CheckInMessageTypeComment) {
			messageType = MessageTypeCheckinComment
			content = message.SupermapCheckinmessage.Content
			if message.CommentID != 0 {
				content = message.SupermapCheckinmessage.Content
			} else {
				content = "该回复不存在"
			}
		} else if message.SupermapCheckinmessage.MessageType == int32(common.CheckInMessageTypeCommentLike) {
			messageType = MessageTypeCheckinCommentLike
			content = message.SupermapCheckinmessage.Content
		} else {
			messageType = ""
			content = message.SupermapCheckinmessage.Content
		}
		var checkinWithImage CheckInWithImage
		if message.SupermapCheckin.ID != 0 {
			checkinWithImage = CheckInWithImage{
				SupermapCheckin: message.SupermapCheckin,
				Images:          _images,
			}
		} else {
			checkinWithImage = CheckInWithImage{
				SupermapCheckin: model.SupermapCheckin{
					Content: "该内容不存在",
				},
				Images: []model.SupermapCheckinimage{},
			}
		}
		messageItems = append(messageItems, MessageItem{
			User:        &message.User,
			CheckIn:     &checkinWithImage,
			Content:     content,
			Timestamp:   message.SupermapCheckinmessage.CreatedAt.Unix(),
			MessageType: messageType,
		})
	}

	if len(messages) > 0 && pageInt == 1 {
		// 更新读取进度：将该用户的最后读取消息ID更新为当前列表中最新的一条；
		// 若不存在记录，则创建一条新的读取进度记录
		err = ec.Nu.DB.
			Model(&model.SupermapCheckinmessagereadprogress{}).
			Where("user_id = ?", ec.AuthData.User.ID).
			Assign(&model.SupermapCheckinmessagereadprogress{
				LastReadID: messages[0].SupermapCheckinmessage.ID,
			}).
			FirstOrCreate(&model.SupermapCheckinmessagereadprogress{
				UserID: ec.AuthData.User.ID,
			}).Error
		if err != nil {
			sdlog.Errorf("[checkinMessageList] 更新读取进度失败: %v", err)
		}
	}

	return webapi.OK(map[string]interface{}{
		"total":     total,
		"messages":  messageItems,
		"page":      pageInt,
		"page_size": pageSizeInt,
	}).Render(ec)
}

type DeleteCheckinRequest struct {
	CheckinID int64 `json:"checkin_id"`
}

func deleteCheckin(ec *middleware.AppRequestContext) error {
	var reqParam DeleteCheckinRequest
	err := ec.Bind(&reqParam)
	if err != nil {
		sdlog.Errorf("[deleteCheckin] 绑定参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// check checkin id is exists
	var checkin model.SupermapCheckin
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// check user is owner of checkin
	if checkin.UserID != ec.AuthData.User.ID {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	// delete checkin
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).Delete(&model.SupermapCheckin{}).Error
	if err != nil {
		sdlog.Errorf("[deleteCheckin] 删除打卡点失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// update checkinpoint checkin_count - 1
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).Where("id = ? and checkin_count > 0", checkin.CheckInPointID).Update("checkin_count", gorm.Expr("checkin_count - 1")).Error
	if err != nil {
		sdlog.Errorf("[deleteCheckin] 更新打卡点数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}

type UpdateCheckinPrivateRequest struct {
	CheckinID int64 `json:"checkin_id"`
	Private   bool  `json:"private"`
}

func updateCheckinPrivate(ec *middleware.AppRequestContext) error {
	var reqParam UpdateCheckinPrivateRequest
	err := ec.Bind(&reqParam)
	if err != nil {
		sdlog.Errorf("[updateCheckinPrivate] 绑定参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// check checkin id is exists
	var checkin model.SupermapCheckin
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).First(&checkin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// check user is owner of checkin
	if checkin.UserID != ec.AuthData.User.ID {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	// update checkin private
	err = ec.Nu.DB.Model(&model.SupermapCheckin{}).Where("id = ?", reqParam.CheckinID).Update("private", reqParam.Private).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}

type LikeCheckinCommentRequest struct {
	CheckinID int64 `json:"checkin_id"`
	CommentID int64 `json:"comment_id"`
}

// POST /api/v1/checkin/comment/like  table: supermap_checkincommentlike
// 先检查comment是否存在
// 再检查用户是否已经点赞
// 如果已经点赞，则取消点赞
// 如果未点赞，则点赞
// 返回是否点赞
func likeCheckinComment(ec *middleware.AppRequestContext) error {
	var reqParam LikeCheckinCommentRequest
	err := ec.Bind(&reqParam)
	if err != nil {
		sdlog.Errorf("[likeCheckinComment] 绑定参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// check checkin id is exists
	var comment model.SupermapCheckincomment
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("id = ? and check_in_id = ?", reqParam.CommentID, reqParam.CheckinID).First(&comment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCheckInCommentNotExists).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// check if user has already liked the comment
	var like model.SupermapCheckincommentlike
	err = ec.Nu.DB.Model(&model.SupermapCheckincommentlike{}).Where("check_in_id = ? and comment_id = ? and user_id = ?", reqParam.CheckinID, reqParam.CommentID, ec.AuthData.User.ID).First(&like).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}
	var liked bool = false
	if like.ID == 0 {
		// 未点赞 则点赞
		err = ec.Nu.DB.Model(&model.SupermapCheckincommentlike{}).Create(&model.SupermapCheckincommentlike{
			CommentID: reqParam.CommentID,
			CheckInID: reqParam.CheckinID,
			UserID:    ec.AuthData.User.ID,
		}).Error
		if err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
		liked = true
	} else {
		// 已点赞 则取消点赞
		err = ec.Nu.DB.Model(&model.SupermapCheckincommentlike{}).Where("check_in_id = ? and comment_id = ? and user_id = ?", reqParam.CheckinID, reqParam.CommentID, ec.AuthData.User.ID).Delete(&model.SupermapCheckincommentlike{}).Error
		if err != nil {
			sdlog.Errorf("[likeCheckinComment] 取消点赞失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
		liked = false
	}

	var likeCount int64 = 0
	if liked {
		likeCount = int64(comment.LikeCount) + 1
	} else {
		if int64(comment.LikeCount) > 0 {
			likeCount = int64(comment.LikeCount) - 1
		} else {
			likeCount = 0
		}
	}
	// 更新comment的like_count - 1
	err = ec.Nu.DB.Model(&model.SupermapCheckincomment{}).Where("id = ? and check_in_id = ?", reqParam.CommentID, reqParam.CheckinID).Update("like_count", likeCount).Error
	if err != nil {
		sdlog.Errorf("[likeCheckinComment] 更新comment点赞数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 插入message table supermap_checkinmessage
	if comment.UserID != ec.AuthData.User.ID && liked {
		_ = ec.Nu.DB.Model(&model.SupermapCheckinmessage{}).Create(&model.SupermapCheckinmessage{
			UserID:      ec.AuthData.User.ID,
			OtherUserID: comment.UserID,
			CheckInID:   reqParam.CheckinID,
			CommentID:   reqParam.CommentID,
			MessageType: int32(common.CheckInMessageTypeCommentLike),
			Content:     "点赞了你的评论",
		}).Error
	}

	return webapi.OK(map[string]bool{
		"liked": liked,
	}).Render(ec)
}

// 非会员最多发布5条，会员无限制
// 检查用户是否是会员
// 检查用户当前发布数量
// 检查用户是否可以发布
type CheckinPublishRestrictResponse struct {
	IsVip               bool  `json:"is_vip"`
	MaxPublishCount     int64 `json:"max_publish_count"`
	CurrentPublishCount int64 `json:"current_publish_count"`
	CanPublish          bool  `json:"can_publish"`
}

func getCheckinPublishRestrict(ec *middleware.AppRequestContext) error {
	restrict, err := db.CheckinPublishRestrict(ec.Nu.DB, ec.AuthData.User.ID)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(CheckinPublishRestrictResponse{
		IsVip:               restrict.IsVip,
		MaxPublishCount:     restrict.MaxPublishCount,
		CurrentPublishCount: restrict.CurrentPublishCount,
		CanPublish:          restrict.CanPublish,
	}).Render(ec)
}
