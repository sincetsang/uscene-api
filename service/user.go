package service

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/service/db"
	"TMA/service/db/model"
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func MaskUsername(username string) string {
	runes := []rune(username)
	length := len(runes)

	if length <= 6 {
		return username
	}

	if at := strings.Index(username, "@"); at != -1 {
		local := []rune(username[:at])
		domain := username[at:]
		if len(local) <= 3 {
			return string(local) + "***" + domain
		}
		return string(local[:3]) + "****" + domain
	}

	// 普通用户名，长度 > 6：中间3位替换为 ***
	mid := length / 2
	start := mid - 1
	if length < 7 {
		start = 1 // 边界防错，最小中间起始位
	}
	end := start + 3

	// 构造脱敏后的字符串
	return string(runes[:start]) + "***" + string(runes[end:])
}

func AddUser(ctx context.Context, nu *nucl.Nucleus, uid int64, provider, userName, avatarUrl, inviteCode, walletAddress, externalId string) error {
	nickname := MaskUsername(userName)
	level := int32(1)
	err := nu.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userLevelInfo := model.UserLevelConfig{}
		err := tx.Model(&model.UserLevelConfig{}).Where("level_id", level).First(&userLevelInfo).Error
		if err != nil {
			return err
		}

		userInfo := model.User{
			UserID:           uid,
			Nickname:         nickname,
			AvatarURL:        avatarUrl,
			InviteCode:       sdrand.String(13, sdrand.LowerLetterNumbers),
			Level:            level,
			HashRate:         userLevelInfo.HashRate,
			Stamina:          userLevelInfo.Stamina,
			RemainingStamina: userLevelInfo.Stamina,
			AvailableSignins: userLevelInfo.CheckinLimit,
			IsPremium:        0,
		}
		err = tx.Model(&model.User{}).Create(&userInfo).Error
		if err != nil {
			return err
		}
		authInfo := model.UserAuth{
			UserID:        uid,
			Provider:      provider,
			WalletAddress: walletAddress,
			ExternalID:    externalId,
		}
		err = tx.Model(&model.UserAuth{}).Create(&authInfo).Error
		if err != nil {
			return err
		}
		inviteInfo := model.User{}
		if inviteCode != "" {
			err = tx.Model(&model.User{}).Where("invite_code", inviteCode).First(&inviteInfo).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil
				}
				return err
			}
			today := time.Now().Format("20060102")
			todayInt, _ := strconv.Atoi(today)
			err = tx.Model(&model.UserInviteRecord{}).Create(&model.UserInviteRecord{
				InviterID: inviteInfo.UserID,
				InviteeID: uid,
				Day:       int32(todayInt),
				Rewards:   0,
			}).Error
			if err != nil {
				return err
			}
		}
		//增加新用户默认等级的奖励
		attrLog := []model.UserAttributeRewardsLog{
			{
				UserID:   uid,
				Category: common.ConfigCategoryLevel,
				Label:    common.ConfigHashRate,
				Rewards:  userLevelInfo.HashRate,
			},
			{
				UserID:   uid,
				Category: common.ConfigCategoryLevel,
				Label:    common.ConfigStamina,
				Rewards:  float64(userLevelInfo.Stamina),
			},
			{
				UserID:   uid,
				Category: common.ConfigCategoryLevel,
				Label:    common.ConfigAvailableSignins,
				Rewards:  float64(userLevelInfo.CheckinLimit),
			},
		}
		err = tx.Model(&model.UserAttributeRewardsLog{}).Create(&attrLog).Error
		return err
	})
	if err != nil {
		sdlog.Warnf("addUser fail,user_id:%d,inviteCode:%s,err:%v", uid, inviteCode, err.Error())
		return common.ErrService
	}
	return err
}

func UserLevelInfo(ctx context.Context, nu *nucl.Nucleus, uid int64) ([]model.UserLevelConfig, error) {
	res := []model.UserLevelConfig{}

	uinfo, err := db.UserInfoById(nu.DB.WithContext(ctx), uid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return res, common.ErrParam
		}
		sdlog.Warn("UserLevelInfo,user_info,uid:%d,err:%+v", uid, err)
		return res, common.ErrParam
	}

	err = nu.DB.WithContext(ctx).Model(&model.UserLevelConfig{}).Where("level_id >= ?", uinfo.Level).Order("level_id asc").Find(&res).Error
	if err != nil {
		sdlog.Warn("UserLevelInfo,level_config,uid:%d,err:%+v", uid, err)
		return res, common.ErrService
	}
	return res, err
}

func UserUpgrade(ctx context.Context, nu *nucl.Nucleus, uid int64) error {

	uinfo, err := db.UserInfoById(nu.DB.WithContext(ctx), uid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return common.ErrParam
		}
		sdlog.Warn("UserLevelInfo,user_info,uid:%d,err:%+v", uid, err)
		return common.ErrParam
	}

	levelInfo := []model.UserLevelConfig{}
	err = nu.DB.WithContext(ctx).Model(&model.UserLevelConfig{}).Where("level_id>=?", uinfo.Level).Order("level_id asc").Limit(2).Find(&levelInfo).Error
	if err != nil {
		return err
	}
	if len(levelInfo) < 2 {
		return common.ErrLastedLevel
	}

	//if uinfo.LandBalance < levelInfo[0].Cost {
	//	return common.ErrBalance
	//}
	//
	//err = nu.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	//	history := model.UserHistory{
	//		UserID:      uid,
	//		UserName:    "",
	//		HistoryType: common.HistoryTypeByUserLevelUpgrade,
	//		OtherSideID: int64(levelInfo[0].Level) + 1,
	//		Reward:      levelInfo[0].Cost,
	//		TokenType:   common.HistoryTokenTypeByPotato,
	//	}
	//	err = tx.Model(&model.UserHistory{}).Create(&history).Error
	//	if err != nil {
	//		return err
	//	}
	//	err = nu.DB.WithContext(ctx).
	//		Model(&model.User{}).
	//		Where("user_id = ?", uid).
	//		Updates(map[string]interface{}{
	//			"land_balance": gorm.Expr("land_balance - ?", int32(levelInfo[0].Cost)),
	//			"level":        levelInfo[0].Level + 1,
	//		}).Error
	//	return err
	//})
	return err
}

func UserTransferLand(ctx context.Context, nu *nucl.Nucleus, senderId, receiveId int64, amount float64) error {

	amount = math.Floor(amount*100) / 100
	err := nu.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		receiveInfo, err := db.UserInfoById(tx, receiveId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return common.ErrUserNotExist
			}
			sdlog.Infof("UserTransferLand,get receiveInfo fail,err:%v", err)
			return common.ErrParam
		}

		senderInfo, err := db.UserInfoById(tx, senderId)
		if err != nil {
			sdlog.Infof("UserTransferLand,get senderInfo fail,err:%v", err)
			return common.ErrService
		}
		if senderInfo.Balance < amount {
			return common.ErrBalance
		}

		err = tx.Model(&model.User{}).Select("balance").Where("user_id = ?", senderId).Updates(map[string]interface{}{
			"balance": gorm.Expr("balance - ?", amount),
		}).Error
		if err != nil {
			sdlog.Warnf("UserTransferLand sub balance,err:%s", err.Error())
			return nil
		}

		err = tx.Model(&model.User{}).Select("balance").Where("user_id = ?", receiveId).Updates(map[string]interface{}{
			"balance": gorm.Expr("balance + ?", amount),
		}).Error
		if err != nil {
			sdlog.Warnf("UserTransferLand add balance,err:%s", err.Error())
			return nil
		}
		record := model.UserTransferRecord{
			SenderID:    senderId,
			SenderName:  senderInfo.Nickname,
			ReceiveID:   receiveId,
			ReceiveName: receiveInfo.Nickname,
			Amount:      amount,
			Fee:         0,
		}
		err = tx.Model(&model.UserTransferRecord{}).Create(&record).Error
		return err
	})
	return err
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func LevelInfoByLevel(ctx context.Context, nu *nucl.Nucleus, level int32) (model.UserLevelConfig, error) {
	res := model.UserLevelConfig{}
	err := nu.DB.WithContext(ctx).Model(&model.UserLevelConfig{}).Where("level >= ?", level).Order("level asc").First(&res).Error
	if err != nil {
		sdlog.Warn("level_config err:%+v", err)
		return res, common.ErrService
	}
	return res, err
}

// create user stat if not exists
func CreateUserStat(ctx context.Context, nu *nucl.Nucleus, uid int64) error {
	err := nu.DB.WithContext(ctx).Model(&model.SupermapUserstat{}).Where("user_id = ?", uid).First(&model.SupermapUserstat{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			stat := model.SupermapUserstat{
				UserID:         uid,
				FollowersCount: 0,
				FollowingCount: int32(0),
				LikesCount:     int32(0),
				CheckinCount:   int32(0),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			err = nu.DB.WithContext(ctx).Model(&model.SupermapUserstat{}).Create(&stat).Error
			if err != nil {
				sdlog.Warn("CreateUserStat,create user_stat,uid:%d,err:%+v", uid, err)
				return common.ErrService
			}
		}
		sdlog.Warn("CreateUserStat,user_stat,uid:%d,err:%+v", uid, err)
		return common.ErrService
	}

	return nil
}
