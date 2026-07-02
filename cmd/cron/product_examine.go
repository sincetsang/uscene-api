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
	productExamineLockKey = "product_examine_lock"
	productExamineLockTTL = 10 * time.Minute
)

func productToExamine(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, productExamineLockKey, lockValue, productExamineLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		fmt.Println("Another instance is already running, exiting...")
		return nil
	}

	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, productExamineLockKey, lockValue); errDefer != nil {
			fmt.Printf("Failed to release lock: %v\n", errDefer)
		} else {
			fmt.Println("lock released")
		}
	}()

	// 1. 查询所有待审核的地标
	var products []model.Product
	createdAt := time.Now().Add(-3 * time.Minute)
	err = nu.DB.WithContext(ctx).
		Where("audit_status = ? AND created_at <= ?", common.SpotAuditStatusPending, createdAt).
		Find(&products).Error
	if err != nil {
		return fmt.Errorf("查询待审核地标失败: %v", err)
	}

	if len(products) == 0 {
		sdlog.Infof("[spotToExamine] 没有待审核的地标")
		productImageToExamine(ctx, nu)
		return nil
	}

	// 2. 遍历每个地标
	for _, product := range products {
		// 查询该地标的所有图片
		var images []model.ProductImage
		err = nu.DB.WithContext(ctx).
			Where("product_id = ?", product.ID).
			Find(&images).Error
		if err != nil {
			sdlog.Errorf("[productToExamine] 查询产品图片失败: product_id=%d, err=%v", product.ID, err)
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
					err = nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
						Where("id = ?", image.ID).
						Update("status", common.UserImageStatusRejected).Error
					if err != nil {
						sdlog.Errorf("[spotToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					}
				} else {
					// 图片内容合规，更新图片状态为通过
					err = nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
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

		err = nu.DB.WithContext(ctx).Model(&model.Product{}).
			Where("id = ?", product.ID).
			Updates(map[string]interface{}{
				"audit_status": newStatus,
				"updated_at":   time.Now(),
			}).Error
		if err != nil {
			sdlog.Errorf("[productToExamine] 更新产品状态失败: product_id=%d, err=%v", product.ID, err)
			continue
		}

		sdlog.Infof("[productToExamine] 产品审核完成: product_id=%d, status=%d", product.ID, newStatus)
	}

	return nil
}

func productImageToExamine(ctx context.Context, nu *nucl.Nucleus) error {
	var productImages []model.ProductImage
	err := nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
		Where("status = ?", common.UserImageStatusPending).
		Find(&productImages).Error
	if err != nil {
		sdlog.Errorf("[productToExamine] 查询产品图片失败: product_images=%v, err=%v", productImages, err)
		return err
	}

	for _, image := range productImages {
		fileKey := strings.TrimPrefix(image.ImageURL, fmt.Sprintf("%s/", nu.Config.Aws.AwsS3CustomDomain))
		if len(fileKey) > 0 {
			svc := s3.New(nu.AwsSession)
			err = service.CheckImageContent(ctx, nu, svc, nu.Config.Aws.AwsStorageBucketName, fileKey)
			if err != nil {
				sdlog.Errorf("[productToExamine] 图片内容违规: image_id=%d, err=%v", image.ID, err)
				err = nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
					Where("id = ?", image.ID).
					Update("status", common.UserImageStatusRejected).Error
				if err != nil {
					sdlog.Errorf("[productToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					return err
				}
			} else {
				err = nu.DB.WithContext(ctx).Model(&model.ProductImage{}).
					Where("id = ?", image.ID).
					Update("status", common.UserImageStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[productToExamine] 更新图片状态失败: image_id=%d, err=%v", image.ID, err)
					return err
				}
				sdlog.Infof("[productToExamine] 产品图片审核完成: image_id=%d, status=%d", image.ID, image.Status)

				// 如果原产品审核状态为待审核，则更新为审核通过
				err = nu.DB.WithContext(ctx).Model(&model.Product{}).
					Where("id = ? && audit_status = ?", image.ProductID, common.SpotAuditStatusPending).
					Update("audit_status", common.SpotAuditStatusApproved).Error
				if err != nil {
					sdlog.Errorf("[productToExamine] 更新产品状态失败: product_id=%d, err=%v", image.ProductID, err)
					return err
				}
				sdlog.Infof("[productToExamine] 产品图片审核完成: image_id=%d, status=%d", image.ID, image.Status)
			}
		}
	}
	return nil
}
