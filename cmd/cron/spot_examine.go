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
	spotExamineLockKey = "spot_examine_lock"
	spotExamineLockTTL = 10 * time.Minute
)

func spotToExamine(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, spotExamineLockKey, lockValue, spotExamineLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, spotExamineLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	// 1. 查询所有待审核的地标
	var spots []model.CheckInPoint
	createdAt := time.Now().Add(-3 * time.Minute)
	err = nu.DB.WithContext(ctx).
		Where("audit_status = ? AND created_at <= ?", common.SpotAuditStatusPending, createdAt).
		Find(&spots).Error
	if err != nil {
		return fmt.Errorf("查询待审核地标失败: %v", err)
	}

	if len(spots) == 0 {
		sdlog.Infof("[spotToExamine] 没有待审核的地标")
		return nil
	}

	// 2. 遍历每个地标
	for _, spot := range spots {
		// 查询该地标的所有图片
		var images []model.CheckInPointImage
		err = nu.DB.WithContext(ctx).
			Where("check_in_point_id = ?", spot.ID).
			Find(&images).Error
		if err != nil {
			sdlog.Errorf("[spotToExamine] 查询地标图片失败: spot_id=%d, err=%v", spot.ID, err)
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
					err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
						Where("id = ?", image.ID).
						Update("status", common.UserImageStatusRejected).Error
					if err != nil {
						sdlog.Errorf("[spotToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					}
				} else {
					// 图片内容合规，更新图片状态为通过
					err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
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
			newStatus = common.SpotAuditStatusApproved
		} else {
			newStatus = common.SpotAuditStatusRejected
		}

		err = nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
			Where("id = ?", spot.ID).
			Updates(map[string]interface{}{
				"audit_status": newStatus,
				"updated_at":   time.Now(),
			}).Error
		if err != nil {
			sdlog.Errorf("[spotToExamine] 更新地标状态失败: spot_id=%d, err=%v", spot.ID, err)
			continue
		}

		sdlog.Infof("[spotToExamine] 地标审核完成: spot_id=%d, status=%d", spot.ID, newStatus)
	}

	var spotImages []model.CheckInPointImage
	err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
		Where("status = ?", common.UserImageStatusPending).
		Find(&spotImages).Error
	if err != nil {
		sdlog.Errorf("[spotToExamine] 查询地标图片失败: spot_images=%v, err=%v", spotImages, err)
	}

	for _, image := range spotImages {
		fileKey := strings.TrimPrefix(image.ImageURL, fmt.Sprintf("%s/", nu.Config.Aws.AwsS3CustomDomain))
		if len(fileKey) > 0 {
			svc := s3.New(nu.AwsSession)
			err = service.CheckImageContent(ctx, nu, svc, nu.Config.Aws.AwsStorageBucketName, fileKey)
			if err != nil {
				sdlog.Errorf("[spotToExamine] 图片内容违规: image_id=%d, err=%v", image.ID, err)
			} else {
				err = nu.DB.WithContext(ctx).Model(&model.CheckInPointImage{}).
					Where("id = ?", image.ID).
					Update("status", common.UserImageStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[spotToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
				}
				sdlog.Infof("[spotToExamine] 地标图片审核完成: image_id=%d, status=%d", image.ID, image.Status)

				// 如果原地标审核状态为待审核，则更新为审核通过
				err = nu.DB.WithContext(ctx).Model(&model.CheckInPoint{}).
					Where("id = ? && audit_status = ?", image.CheckInPointID, common.SpotAuditStatusPending).
					Update("audit_status", common.SpotAuditStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[spotToExamine] 更新地标状态失败: spot_id=%d, err=%v", image.CheckInPointID, err)
				}
			}
		}
	}

	return nil
}
