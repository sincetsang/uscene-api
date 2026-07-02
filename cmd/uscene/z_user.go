package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db"
	"TMA/service/db/model"
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

type UserInfoRes struct {
	User            model.User                                 `json:"user"`
	HashRateRanking int64                                      `json:"hash_rate_ranking"`
	StaminaRanking  int64                                      `json:"stamina_ranking"`
	SignINSRanking  int64                                      `json:"sign_ins_ranking"`
	UserAttrLog     map[string][]model.UserAttributeRewardsLog `json:"user_attr_log"`
	UserAuths       []model.UserAuth                           `json:"user_auths"`  // 用户绑定的第三方账号信息
	UserAssets      []UserAssetRes                             `json:"user_assets"` // 用户资产
	UserStat        []model.SupermapUserstat                   `json:"user_stat"`   // 用户打卡点统计
}

type UserAssetRes struct {
	Currency string          `json:"currency"` // 货币
	Balance  decimal.Decimal `json:"balance"`  // 余额
}

func userInfo(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	uInfo, err := db.UserInfoById(ec.Nu.DB, userId)
	if err != nil {
		sdlog.Errorf("[userInfo] 查询用户信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	HashRateRanking := int64(1)
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("hash_rate > ?", uInfo.HashRate).Count(&HashRateRanking).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询哈希率排名失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if HashRateRanking == 0 {
		HashRateRanking = 1
	}
	StaminaRanking := int64(1)
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("stamina > ?", uInfo.Stamina).Count(&StaminaRanking).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询耐力排名失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if StaminaRanking == 0 {
		StaminaRanking = 1
	}
	SignINSRanking := int64(1)
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("available_signins > ?", uInfo.AvailableSignins).Count(&SignINSRanking).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询签到排名失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	if SignINSRanking == 0 {
		SignINSRanking = 1
	}
	attrLog := []model.UserAttributeRewardsLog{}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserAttributeRewardsLog{}).Where("user_id", uInfo.UserID).Find(&attrLog).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询用户属性日志失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	attrLogMap := map[string][]model.UserAttributeRewardsLog{}
	for _, log := range attrLog {
		attrLogMap[log.Label] = append(attrLogMap[log.Label], log)
	}

	// 获取用户第三方账号信息
	var userAuths []model.UserAuth
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", userId).
		Find(&userAuths).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询用户第三方账号信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取用户资产信息
	var userAssets []model.UserAsset
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", userId).
		Find(&userAssets).Error
	if err != nil {
		sdlog.Errorf("[userInfo] 查询用户资产信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 构建资产映射
	assetMap := []UserAssetRes{}
	for _, asset := range userAssets {
		assetMap = append(assetMap, UserAssetRes{
			Currency: asset.Currency,
			Balance:  decimal.NewFromFloat(asset.Balance),
		})
	}

	// user stat
	var userStat []model.SupermapUserstat
	_ = ec.Nu.DB.Model(&model.SupermapUserstat{}).Where("user_id = ?", userId).Find(&userStat).Error

	return webapi.OK(UserInfoRes{
		User:            *uInfo,
		HashRateRanking: HashRateRanking,
		StaminaRanking:  StaminaRanking,
		SignINSRanking:  SignINSRanking,
		UserAttrLog:     attrLogMap,
		UserAuths:       userAuths,
		UserAssets:      assetMap,
		UserStat:        userStat,
	}).Render(ec)
}

type OtherUserInfoRes struct {
	User       model.User               `json:"user"`
	UserStat   []model.SupermapUserstat `json:"user_stat"`   // 用户打卡点统计
	IsFollowed bool                     `json:"is_followed"` // 是否已关注
}

// get请求， user id 为其他用户的 id
func getUserInfoOther(ec *middleware.AppRequestContext) error {
	userId := ec.Request().URL.Query().Get("user_id")
	if userId == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userIdInt, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	uInfo, err := db.UserInfoById(ec.Nu.DB, userIdInt)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrUserNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	var userStat []model.SupermapUserstat
	_ = ec.Nu.DB.Model(&model.SupermapUserstat{}).Where("user_id = ?", userIdInt).Find(&userStat).Error

	isFollowed := false
	err = ec.Nu.DB.Model(&model.SupermapUserfollow{}).Where("user_id = ? and followed_user_id = ?", ec.AuthData.User.ID, userIdInt).First(&model.SupermapUserfollow{}).Error
	if err != nil {
		sdlog.Errorf("[getUserInfoOther] 查询用户关注信息失败: %v", err)
		if err == gorm.ErrRecordNotFound {
			isFollowed = false
		}
	} else {
		isFollowed = true
	}
	return webapi.OK(OtherUserInfoRes{
		User:       *uInfo,
		UserStat:   userStat,
		IsFollowed: isFollowed,
	}).Render(ec)
}

type SearchParam struct {
	KeyWord string `json:"key_word"`
}
type SearchRes struct {
	UserId    int64  `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarUrl string `json:"avatar_url"`
}

func userSearch(ec *middleware.AppRequestContext) error {
	req := SearchParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	keyword := req.KeyWord + "%"
	list := []model.User{}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("user_id LIKE ? or nickname like ?", keyword, keyword).Limit(10).Find(&list).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrUserNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	res := []SearchRes{}
	for _, info := range list {
		res = append(res, SearchRes{
			UserId:    info.UserID,
			UserName:  info.Nickname,
			AvatarUrl: common.UserAvatarURL + info.AvatarURL,
		})
	}
	return webapi.OK(res).Render(ec)
}

type TransferParam struct {
	ReceiveId int64   `json:"receive_id"`
	Amount    float64 `json:"amount"`
}

func userTransfer(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := TransferParam{}
	err := ec.Bind(&req)
	if err != nil || req.Amount <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.ReceiveId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Amount <= 0 {
		return webapi.Error(common.ErrUserTransferAmountLimit).Render(ec)
	}
	if userId == req.ReceiveId {
		return webapi.Error(common.ErrUserTransferToYourself).Render(ec)
	}
	lockStr := "transfer:" + strconv.FormatInt(userId, 10)
	ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, lockStr)
	if !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}
	err = service.UserTransferLand(ec.Request().Context(), ec.Nu, userId, req.ReceiveId, req.Amount)
	if err != nil {
		sdlog.Errorf("[userTransfer] 用户转账失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}

func userTransferRecord(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	list := []model.UserTransferRecord{}
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserTransferRecord{}).Where("sender_id=? or receive_id=?", userId, userId).Order("id desc").Find(&list).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(list).Render(ec)
}

type UserLatLngParam struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type UserLatLngParamResp struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func updateUserLatLng(ec *middleware.AppRequestContext) error {
	req := UserLatLngParam{}
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	updateUser := &model.User{
		Latitude:  req.Lat,
		Longitude: req.Lng,
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Updates(updateUser).Error
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	return webapi.OK(req).Render(ec)
}

type UserIncomeRes struct {
	User          model.User    `json:"user"`
	MoveIncome    float64       `json:"move_income"`
	CheckInIncome float64       `json:"check_in_income"`
	CheckInDetail []CheckInSpot `json:"check_in_detail"`
}

// CheckInSpot 签到点收入详情
type CheckInSpot struct {
	SpotName string  `json:"spot_name"`
	Income   float64 `json:"income"`
}

func userIncome(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	uInfo, err := db.UserInfoById(ec.Nu.DB, userId)
	if err != nil {
		sdlog.Errorf("[userIncome] 查询用户信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取今天的起始时间和结束时间
	startOfDay := time.Now().Truncate(24 * time.Hour)            // 今天00:00:00
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second) // 今天23:59:59

	// 使用 sql.NullFloat64 来处理 NULL 值
	var checkInIncomeNull struct {
		Income float64 `gorm:"column:income"`
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Select("COALESCE(SUM(rewards), 0) as income").
		Where("user_id = ? AND category = ? AND created_at BETWEEN ? AND ?",
			userId, common.ConfigAssetTypeByCheckIn, startOfDay, endOfDay).
		Scan(&checkInIncomeNull).Error
	if err != nil {
		sdlog.Errorf("[userIncome] 查询签到收入失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	var moveIncomeNull struct {
		Income float64 `gorm:"column:income"`
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Select("COALESCE(SUM(rewards), 0) as income").
		Where("user_id = ? AND category = ? AND created_at BETWEEN ? AND ?",
			userId, common.ConfigAssetTypeByMove, startOfDay, endOfDay).
		Scan(&moveIncomeNull).Error
	if err != nil {
		sdlog.Errorf("[userIncome] 查询移动收入失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取今天的签到记录和对应的打卡点信息
	checkInDetail := []CheckInSpot{}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.UserAssetDetail{}).
		Select("check_in_point.title as spot_name, user_asset_detail.rewards as income").
		Joins("LEFT JOIN check_in_point ON user_asset_detail.related_id = check_in_point.id").
		Where("user_asset_detail.user_id = ? AND user_asset_detail.category = ? AND user_asset_detail.created_at BETWEEN ? AND ?",
			userId, common.ConfigAssetTypeByCheckIn, startOfDay, endOfDay).
		Order("user_asset_detail.created_at DESC").
		Scan(&checkInDetail).Error
	if err != nil {
		sdlog.Errorf("[userIncome] 查询签到收入失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	attrLog := []model.UserAttributeRewardsLog{}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserAttributeRewardsLog{}).Where("user_id", uInfo.UserID).Find(&attrLog).Error
	if err != nil {
		sdlog.Errorf("[userIncome] 查询用户属性日志失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(UserIncomeRes{
		User:          *uInfo,
		MoveIncome:    moveIncomeNull.Income,
		CheckInIncome: checkInIncomeNull.Income,
		CheckInDetail: checkInDetail,
	}).Render(ec)
}

// 设置密码请求参数
type SetPasswordParam struct {
	OldPassword     string `json:"old_password,omitempty"` // 旧密码（如果已设置密码则必填）
	NewPassword     string `json:"new_password"`           // 新密码
	ConfirmPassword string `json:"confirm_password"`       // 确认新密码
}

// 设置密码
func setPassword(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	req := SetPasswordParam{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 验证新密码
	if len(req.NewPassword) < 6 {
		return webapi.Error(common.ErrPasswordTooWeak).Render(ec)
	}

	// 确认密码必须匹配
	if req.NewPassword != req.ConfirmPassword {
		return webapi.Error(common.ErrPasswordNotMatch).Render(ec)
	}

	// 查询用户
	var user model.User
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", userId).
		First(&user).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 如果已设置密码，需要验证旧密码
	if user.Password != "" {
		if req.OldPassword == "" {
			return webapi.Error(common.ErrOldPasswordRequired).Render(ec)
		}
		// 验证旧密码
		if !service.CheckPassword(req.OldPassword, user.Password) {
			return webapi.Error(common.ErrOldPasswordIncorrect).Render(ec)
		}
	}

	// 加密新密码
	hashedPassword, err := service.HashPassword(req.NewPassword)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新密码
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.User{}).
		Where("user_id = ?", userId).
		Update("password", hashedPassword).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

// 密码登录请求参数
type PasswordLoginParam struct {
	Phone    string `json:"phone"`    // 手机号
	Password string `json:"password"` // 密码
}

// 手机号密码登录
func passwordLogin(ec *middleware.AppRequestContext) error {
	req := PasswordLoginParam{}
	if err := ec.Bind(&req); err != nil || req.Phone == "" || req.Password == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 先通过手机号在 user_auth 表中查找用户ID
	var uAuth model.UserAuth
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("external_id = ? AND provider = ?", req.Phone, common.ConfigProviderByPhone).
		First(&uAuth).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrUserNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 再通过用户ID查找用户信息和密码
	var user model.User
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Where("user_id = ?", uAuth.UserID).
		First(&user).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 检查是否设置了密码
	if user.Password == "" || !service.CheckPassword(req.Password, user.Password) {
		return webapi.Error(common.ErrPasswordIncorrect).Render(ec)
	}

	// 生成 JWT token
	token, err := webapi.GenerateTokenString([]byte(ec.Nu.Config.WalletJwtSecret), user.UserID, uAuth.WalletAddress, webapi.TokenTypeRefresh, webapi.RefreshTokenDuration)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(token).Render(ec)
}

// 图片上传请求参数
type UploadImageRequest struct {
	Image []byte `json:"image"` // 图片二进制数据
}

// 图片上传响应
type UploadImageResponse struct {
	Id  int64  `json:"id"`
	URL string `json:"url"` // 图片访问URL
}

// 上传图片
func uploadImage(ec *middleware.AppRequestContext) error {
	startTime := time.Now()
	log.Printf("[uploadImage] 开始处理图片上传")

	// 检查用户今日上传图片数量
	userId := ec.AuthData.User.ID
	today := time.Now().Truncate(24 * time.Hour)
	var count int64
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserImage{}).
		Where("user_id = ? AND created_at >= ?", userId, today).
		Count(&count).Error
	if err != nil {
		log.Printf("[uploadImage] 查询用户图片数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	if count >= 200 {
		log.Printf("[uploadImage] 用户今日上传图片数量已达上限: %d", count)
		return webapi.Error(common.ErrUploadLimit).Render(ec)
	}

	// 获取上传的文件
	file, err := ec.FormFile("image")
	if err != nil {
		log.Printf("[uploadImage] 获取文件失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}
	log.Printf("[uploadImage] 获取文件耗时: %v", time.Since(startTime))

	// 获取并检查文件后缀
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	if !allowedExts[ext] {
		log.Printf("[uploadImage] 不支持的文件类型: %s", ext)
		return webapi.Error(common.ErrUnsupportedFileType).Render(ec)
	}

	// 检查文件大小（限制为2MB）
	if file.Size > 2*1024*1024 {
		log.Printf("[uploadImage] 文件大小超限: %d", file.Size)
		return webapi.Error(common.ErrFileSizeLimit).Render(ec)
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		log.Printf("[uploadImage] 打开文件失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	defer src.Close()

	// 读取文件内容
	readStart := time.Now()
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		log.Printf("[uploadImage] 读取文件失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	log.Printf("[uploadImage] 读取文件耗时: %v", time.Since(readStart))

	// 使用 AWS Rekognition 检测图片内容
	// rekognitionStart := time.Now()
	// moderationInput := &rekognition.DetectModerationLabelsInput{
	// 	Image: &rekognition.Image{
	// 		Bytes: fileBytes,
	// 	},
	// 	MinConfidence: aws.Float64(80.0),
	// }

	// result, err := ec.Nu.AwsClient.DetectModerationLabels(moderationInput)
	// if err != nil {
	// 	if _, ok := err.(*rekognition.InvalidImageFormatException); ok {
	// 		log.Printf("[uploadImage] 图片格式无效: %v", err)
	// 		return webapi.Error(common.ErrUnsupportedFileType).Render(ec)
	// 	}
	// 	log.Printf("[uploadImage] AWS Rekognition检测失败: %v", err)
	// 	return webapi.Error(common.ErrService).Render(ec)
	// }
	// log.Printf("[uploadImage] AWS Rekognition检测耗时: %v", time.Since(rekognitionStart))

	// 如果检测到不适合的内容，拒绝上传
	// if len(result.ModerationLabels) > 0 {
	// 	log.Printf("[uploadImage] 检测到不适合的内容: %v", result.ModerationLabels)
	// 	return webapi.Error(common.ErrImageContentInappropriate).Render(ec)
	// }

	// 生成唯一文件名
	fileName := fmt.Sprintf("user/%d_%s%s",
		ec.AuthData.User.ID,
		time.Now().Format("20060102150405"),
		ext)

	// 上传到S3
	s3Start := time.Now()
	svc := s3.New(ec.Nu.AwsSession)
	input := &s3.PutObjectInput{
		Bucket:      aws.String(ec.Nu.Config.Aws.AwsStorageBucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String("image/" + ext),
	}

	_, err = svc.PutObject(input)
	if err != nil {
		log.Printf("[uploadImage] 上传到S3失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}
	log.Printf("[uploadImage] 上传到S3耗时: %v", time.Since(s3Start))

	imageURL := fmt.Sprintf("%s/%s", ec.Nu.Config.Aws.AwsS3CustomDomain, fileName)

	// 将图片信息存储到user_image表
	userImage := model.UserImage{
		UserID:    ec.AuthData.User.ID,
		SpotID:    0,
		Status:    1,
		ImageURL:  imageURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := ec.Nu.DB.WithContext(ec.Request().Context()).Create(&userImage).Error; err != nil {
		sdlog.WithError(err).Error("[uploadImage] 保存图片信息失败")
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 返回 CDN URL
	log.Printf("[uploadImage] 总耗时: %v", time.Since(startTime))
	return webapi.OK(UploadImageResponse{
		Id:  userImage.ID,
		URL: imageURL,
	}).Render(ec)
}

// 我的地标列表响应
type MySpotListResponse struct {
	TotalCount int64                    `json:"total_count"`
	Points     []*CheckInPointWithOwner `json:"points"`
}

type CheckInPointWithOwner struct {
	Point                  *model.CheckInPoint              `json:"point"`
	OwnerUsername          string                           `json:"owner_username"`
	OwnerAvatar            string                           `json:"owner_avatar"`
	Liked                  bool                             `json:"liked"`
	CategoryImage          string                           `json:"category_image"`
	Images                 []model.CheckInPointImage        `gorm:"-"  json:"images"`
	Claim                  float64                          `json:"claim"`
	Product                []model.Product                  `gorm:"-"  json:"product"`
	Advertisement          *model.CheckInPointAdvertisement `json:"advertisement" gorm:"check_in_point_advertisement"` // 广告详情
	AdvertisementIsClaimed bool                             `json:"advertisement_is_claimed" gorm:"-"`                 // 广告是否已领取
}

// 获取我的地标列表
func mySpotList(ec *middleware.AppRequestContext) error {

	userId := ec.AuthData.User.ID

	// 获取状态参数，默认为已激活(1)
	status := ec.QueryParams().Get("status")
	keyword := ec.QueryParams().Get("keyword")
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "1000"
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if status == "" {
		status = "1"
	}

	if pageInt <= 0 {
		pageInt = 1
	}
	if pageSizeInt <= 0 {
		pageSizeInt = 1000
	}

	// 定义一个临时结构体来接收基础查询结果
	type BaseResult struct {
		model.CheckInPoint
		Ownername   string `json:"owner_name"`
		OwnerAvatar string `json:"owner_avatar"`
	}

	var count int64
	var baseResults []BaseResult

	// 获取总数
	countQuery := ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.CheckInPoint{}).
		Where("owner_id = ? AND status = ? and is_active=1", userId, status)

	// 添加关键词搜索条件
	if keyword != "" {
		countQuery = countQuery.Where("title LIKE ?", "%"+keyword+"%")
	}

	err = countQuery.Count(&count).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取地标列表
	query := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("check_in_point").
		Select("check_in_point.*, user.nickname as owner_name, user.avatar_url as owner_avatar").
		Joins("LEFT JOIN user ON check_in_point.owner_id = user.user_id").
		Where("check_in_point.owner_id = ? AND check_in_point.status = ? and check_in_point.is_active=1", userId, status)

	// 添加关键词搜索条件
	if keyword != "" {
		query = query.Where("check_in_point.title LIKE ?", "%"+keyword+"%")
	}

	err = query.Order("check_in_point.created_at DESC").
		Limit(pageSizeInt).
		Offset((pageInt - 1) * pageSizeInt).
		Find(&baseResults).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	if len(baseResults) == 0 {
		return webapi.OK(MySpotListResponse{
			TotalCount: 0,
			Points:     []*CheckInPointWithOwner{},
		}).Render(ec)
	}

	// 收集地标ID
	spotIds := make([]int64, len(baseResults))
	for i, spot := range baseResults {
		spotIds[i] = spot.ID
	}

	// 批量查询图片
	var images []model.CheckInPointImage
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.CheckInPointImage{}).
		Where("check_in_point_id IN ?", spotIds).
		Order("check_in_point_id, sort_index").
		Find(&images).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建图片映射
	spotImages := make(map[int64][]model.CheckInPointImage)
	for _, img := range images {
		spotImages[img.CheckInPointID] = append(spotImages[img.CheckInPointID], img)
	}

	// 获取所有分类信息
	var categories []*model.SpotCategory
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.SpotCategory{}).Find(&categories).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	categoryMap := make(map[string]*model.SpotCategory)
	for _, cat := range categories {
		categoryMap[cat.Type] = cat
		categoryMap[cat.Label] = cat
		categoryMap[cat.LabelZh] = cat
		categoryMap[cat.LabelZhTw] = cat
	}
	// 构建最终结果
	points := make([]*CheckInPointWithOwner, len(baseResults))
	for i, base := range baseResults {
		spot := &CheckInPointWithOwner{
			Point:         &base.CheckInPoint,
			OwnerUsername: base.Ownername,
			OwnerAvatar:   base.OwnerAvatar,
			CategoryImage: categoryMap[base.Category].ImageURL,
			Images:        make([]model.CheckInPointImage, 0),
		}

		claim, _ := redis.GetClaim(ec.Request().Context(), ec.Nu, userId, fmt.Sprint(spot.Point.ID))
		spot.Claim = claim
		// 添加其他图片
		if imgs, exists := spotImages[base.ID]; exists {
			spot.Images = append(spot.Images, imgs...)
		}
		// 添加主图
		if base.LandmarkImage != "" && len(spot.Images) == 0 {
			spot.Images = append(spot.Images, model.CheckInPointImage{
				ID:             0,
				CheckInPointID: base.ID,
				ImageURL:       base.LandmarkImage,
				SortIndex:      0,
				Status:         1,
			})
		}
		lan := common.GetRequestLanguage()
		if lan == common.LanguageZh {
			spot.Point.Category = categoryMap[base.Category].LabelZh
		} else if lan == common.LanguageZhTw {
			spot.Point.Category = categoryMap[base.Category].LabelZhTw
		} else if lan == common.LanguageRu {
			spot.Point.Category = categoryMap[base.Category].LabelRu
		}
		points[i] = spot
	}

	return webapi.OK(MySpotListResponse{
		TotalCount: count,
		Points:     points,
	}).Render(ec)
}

// 更新用户信息请求参数
type UpdateUserProfileRequest struct {
	Nickname  string `json:"nickname"`   // 昵称
	AvatarUrl string `json:"avatar_url"` // 头像URL
}

// 更新用户信息
func updateUserProfile(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 1. 解析请求参数
	req := UpdateUserProfileRequest{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 2. 参数验证
	if req.Nickname == "" && req.AvatarUrl == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 3. 构建更新数据
	updates := map[string]interface{}{}
	if req.Nickname != "" {
		if len(req.Nickname) > 32 { // 限制昵称长度
			return webapi.Error(common.ErrParam).Render(ec)
		}
		updates["nickname"] = req.Nickname
	}
	if req.AvatarUrl != "" {
		updates["avatar_url"] = req.AvatarUrl
	}
	updates["updated_at"] = time.Now()

	// 4. 更新数据库
	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.User{}).
		Where("user_id = ?", userId).
		Updates(updates).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

// 核销码列表响应结构体
type MyVerificationCodeListResponse struct {
	TotalCount int64                  `json:"total_count"` // 总数
	Codes      []VerificationCodeInfo `json:"codes"`       // 核销码列表
}

// 核销码信息结构体
type VerificationCodeInfo struct {
	ID          int64     `json:"id"`           // 核销码ID
	Code        string    `json:"code"`         // 核销码
	Status      string    `json:"status"`       // 状态：unused-未使用，used-已使用
	UseTime     time.Time `json:"use_time"`     // 使用时间
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	OrderID     int64     `json:"order_id"`     // 订单ID
	ProductName string    `json:"product_name"` // 产品名称
	SpotID      int64     `json:"spot_id"`      // 打卡点ID
	SpotName    string    `json:"spot_name"`    // 打卡点名称（仅已使用状态时返回）
}

// 获取我的核销码列表
func myVerificationCodeList(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	// 获取状态参数
	status := ec.QueryParams().Get("status")

	// 构建查询条件
	query := ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.VerificationCode{}).
		Select("verification_code.*, verification_code_order.product_name, check_in_point.id as spot_id, check_in_point.title as spot_name").
		Joins("LEFT JOIN verification_code_order ON verification_code.order_id = verification_code_order.id").
		Joins("LEFT JOIN check_in_point ON verification_code.code = check_in_point.verification_code").
		Where("verification_code.buy_uid = ?", userId)

	switch status {
	case common.VerificationCodeStatusUnused:
		query = query.Where("verification_code.status = ?", common.VerificationCodeStatusUnused)
	case common.VerificationCodeStatusUsed:
		query = query.Where("verification_code.status = ?", common.VerificationCodeStatusUsed)
	}

	// 获取总数
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取核销码列表
	type CodeWithProduct struct {
		model.VerificationCode
		ProductName string `gorm:"column:product_name"`
		SpotName    string `gorm:"column:spot_name"`
		SpotID      int64  `gorm:"column:spot_id"`
	}
	var codes []CodeWithProduct
	err = query.Order("verification_code.created_at DESC").Find(&codes).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 构建响应数据
	codeInfos := make([]VerificationCodeInfo, len(codes))
	for i, code := range codes {
		codeInfos[i] = VerificationCodeInfo{
			ID:          code.ID,
			Code:        code.Code,
			Status:      code.Status,
			UseTime:     code.UseTime,
			CreatedAt:   code.CreatedAt,
			OrderID:     code.OrderID,
			ProductName: code.ProductName,
			SpotID:      code.SpotID,
			SpotName:    code.SpotName,
		}
	}

	return webapi.OK(MyVerificationCodeListResponse{
		TotalCount: total,
		Codes:      codeInfos,
	}).Render(ec)
}

type UserRewardsRes struct {
	Total float64 `json:"total"`
	Claim float64 `json:"claim"`
}

func userRewards(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	totalClaimed, err := redis.GetTotalClaimed(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	waitClaim, err := redis.GetTotalClaim(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(UserRewardsRes{
		Total: totalClaimed + waitClaim,
		Claim: waitClaim,
	}).Render(ec)

}

type ClaimRes struct {
	Claim float64 `json:"claim"`
}

func userClaim(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	claimed, err := redis.ClaimAll(ec.Request().Context(), ec.Nu, userId)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(ClaimRes{
		Claim: claimed,
	}).Render(ec)

}

type UserStatisticsDataRes struct {
	TotalSpot                  int64 `json:"total_spot"`
	TotalSpotLiked             int64 `json:"total_spot_liked"`
	TotalProductVisited        int64 `json:"total_product_visited"`
	TotalProductFavorited      int64 `json:"total_product_favorited"`
	TotalProduct               int64 `json:"total_product"`
	TotalVerifyCode            int64 `json:"total_verify_code"`
	TotalVerifyCodeOrder       int64 `json:"total_verify_code_order"`
	TotalProductSold           int64 `json:"total_product_sold"`
	TotalProductBought         int64 `json:"total_product_bought"`
	TotalProductPendingComment int64 `json:"total_product_pending_comment"`
}

func getUserStatisticsData(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	var totalSpot int64
	var totalSpotLiked int64
	var totalProductVisited int64
	var totalProductFavorited int64
	var totalProduct int64
	var totalVerifyCode int64
	var totalVerifyCodeOrder int64
	var totalProductSold int64
	var totalProductBought int64
	var totalProductPendingComment int64

	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.CheckInPoint{}).Where("owner_id = ?", userId).Count(&totalSpot).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserLike{}).Where("user_id = ?", userId).Count(&totalSpotLiked).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductVisitHistory{}).Where("user_id = ?", userId).Count(&totalProductVisited).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductFavorite{}).Where("user_id = ?", userId).Count(&totalProductFavorited).Error

	// 通过 check_in_point 的 owner_id 关联查询用户的产品总数
	err = ec.Nu.DB.WithContext(ec.Request().Context()).
		Model(&model.Product{}).
		Joins("JOIN check_in_point ON check_in_point.id = product.check_in_point_id").
		Where("check_in_point.owner_id = ?", userId).
		Count(&totalProduct).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.VerificationCode{}).Where("buy_uid = ?", userId).Count(&totalVerifyCode).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.VerificationCodeOrder{}).Where("user_id = ?", userId).Count(&totalVerifyCodeOrder).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	validStatus := []string{common.ProductOrderStatusCompleted,
		common.ProductOrderStatusPendingReceipt,
		common.ProductOrderStatusPendingShipment}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("seller_id = ?", userId).Where("status in (?)", validStatus).Count(&totalProductSold).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	validStatus = append(validStatus, common.ProductOrderStatusPendingPayment, common.ProductOrderStatusPaymentProcessing)
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("user_id = ?", userId).Where("status in (?)", validStatus).Count(&totalProductBought).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("user_id = ?", userId).
		Where("status in (?)", []string{common.ProductOrderStatusCompleted}).
		Where("is_commented = ?", false).
		Count(&totalProductPendingComment).Error
	if err != nil {
		sdlog.Errorf("getUserStatisticsData: %v", err)
	}

	return webapi.OK(UserStatisticsDataRes{
		TotalSpot:                  totalSpot,
		TotalSpotLiked:             totalSpotLiked,
		TotalProductVisited:        totalProductVisited,
		TotalProductFavorited:      totalProductFavorited,
		TotalProduct:               totalProduct,
		TotalVerifyCode:            totalVerifyCode,
		TotalVerifyCodeOrder:       totalVerifyCodeOrder,
		TotalProductSold:           totalProductSold,
		TotalProductBought:         totalProductBought,
		TotalProductPendingComment: totalProductPendingComment,
	}).Render(ec)
}

type UpdateUserDescriptionRequest struct {
	Description string `json:"description"`
}

// 修改描述
func updateUserDescription(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID

	req := UpdateUserDescriptionRequest{}
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	description := req.Description
	if description == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.User{}).Where("user_id = ?", userId).Update("description", description).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}
