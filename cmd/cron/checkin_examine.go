package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/service"
	"TMA/service/db/model"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	checkinExamineLockKey = "checkin_examine_lock"
	checkinExamineLockTTL = 10 * time.Minute
)

func checkinToExamine(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, checkinExamineLockKey, lockValue, checkinExamineLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, checkinExamineLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	// 1. 查询所有待审核的地标
	var checkins []model.SupermapCheckin
	err = nu.DB.WithContext(ctx).
		Where("status = ?", common.CheckInStatusPending).
		Find(&checkins).Error
	if err != nil {
		return fmt.Errorf("查询待审核打卡失败: %v", err)
	}

	if len(checkins) == 0 {
		sdlog.Infof("[checkinToExamine] 没有待审核的打卡")
		return nil
	}

	// 2. 遍历每个地标
	for _, checkin := range checkins {
		// 查询该地标的所有图片
		var images []model.SupermapCheckinimage
		err = nu.DB.WithContext(ctx).
			Where("check_in_id = ?", checkin.ID).
			Find(&images).Error
		if err != nil {
			sdlog.Errorf("[checkinToExamine] 查询打卡图片失败: checkin_id=%d, err=%v", checkin.ID, err)
			continue
		}

		// 3. 检查每张图片
		hasApprovedImage := false
		for _, image := range images {
			if image.Status == common.UserImageStatusApproved {
				hasApprovedImage = true
				continue
			}
			// 从图片URL中提取文件key
			fileKey := strings.TrimPrefix(image.ImageURL, fmt.Sprintf("%s/", nu.Config.Aws.AwsS3CustomDomain))
			if len(fileKey) > 0 {
				// 调用图片内容检查
				svc := s3.New(nu.AwsSession)
				err = service.CheckImageContent(ctx, nu, svc, nu.Config.Aws.AwsStorageBucketName, fileKey)
				if err != nil {
					// 图片内容违规，更新图片状态为拒绝
					err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckinimage{}).
						Where("id = ?", image.ID).
						Update("status", common.UserImageStatusRejected).Error
					if err != nil {
						sdlog.Errorf("[spotToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					}
				} else {
					// 图片内容合规，更新图片状态为通过
					err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckinimage{}).
						Where("id = ?", image.ID).
						Update("status", common.UserImageStatusApproved).Error
					if err != nil {
						sdlog.Errorf("[spotToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					}
					hasApprovedImage = true
				}
			}
		}

		// 4. 更新地标审核状态
		var newStatus int32
		if hasApprovedImage {
			newStatus = common.CheckInStatusApproved
		} else {
			newStatus = common.CheckInStatusRejected
		}

		err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckin{}).
			Where("id = ?", checkin.ID).
			Updates(map[string]interface{}{
				"status":     newStatus,
				"updated_at": time.Now(),
			}).Error
		if err != nil {
			sdlog.Errorf("[checkinToExamine] 更新打卡状态失败: checkin_id=%d, err=%v", checkin.ID, err)
			continue
		}

		sdlog.Infof("[checkinToExamine] 打卡审核完成: checkin_id=%d, status=%d", checkin.ID, newStatus)
	}

	var spotImages []model.SupermapCheckinimage
	err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckinimage{}).
		Where("status = ?", common.UserImageStatusPending).
		Find(&spotImages).Error
	if err != nil {
		sdlog.Errorf("[checkinToExamine] 查询打卡图片失败: spot_images=%v, err=%v", spotImages, err)
	}

	for _, image := range spotImages {
		fileKey := strings.TrimPrefix(image.ImageURL, fmt.Sprintf("%s/", nu.Config.Aws.AwsS3CustomDomain))
		if len(fileKey) > 0 {
			svc := s3.New(nu.AwsSession)
			err = service.CheckImageContent(ctx, nu, svc, nu.Config.Aws.AwsStorageBucketName, fileKey)
			if err != nil {
				sdlog.Errorf("[checkinToExamine] 图片内容违规: image_id=%d, err=%v", image.ID, err)
			} else {
				err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckinimage{}).
					Where("id = ?", image.ID).
					Update("status", common.UserImageStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[checkinToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
				}
				sdlog.Infof("[checkinToExamine] 打卡图片审核完成: image_id=%d, status=%d", image.ID, image.Status)

				// 如果原地标审核状态为待审核，则更新为审核通过
				err = nu.DB.WithContext(ctx).Model(&model.SupermapCheckin{}).
					Where("id = ? && status = ?", image.CheckInID, common.CheckInStatusPending).
					Update("status", common.CheckInStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[checkinToExamine] 更新地标状态失败: checkin_id=%d, err=%v", image.CheckInID, err)
				}
			}
		}
	}

	return nil
}
