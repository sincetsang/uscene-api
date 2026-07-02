package service

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/service/db/model"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"gorm.io/gorm"
)

// 验证文件是否已上传（带重试）
func VerifyFileUpload(ctx context.Context, n *nucl.Nucleus, svc *s3.S3, bucket, key string) error {
	maxRetries := 3
	retryDelay := time.Second

	for i := range maxRetries {
		_, err := svc.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err == nil {
			return nil
		}

		// 如果是最后一次重试，直接返回错误
		if i == maxRetries-1 {
			return fmt.Errorf("验证文件上传失败: %v", err)
		}

		// 等待一段时间后重试
		time.Sleep(retryDelay)
		retryDelay *= 2 // 指数退避
	}

	return nil
}

// 检查图片内容是否违规
func CheckImageContent(ctx context.Context, n *nucl.Nucleus, svc *s3.S3, bucket, key string) error {
	// 创建 Rekognition 客户端
	rekognitionSvc := rekognition.New(n.AwsSession)

	// 获取图片内容
	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("获取图片内容失败: %v", err)
	}
	defer result.Body.Close()

	// 读取图片内容
	imageBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("读取图片内容失败: %v", err)
	}

	// 调用 Rekognition 进行内容检测
	input := &rekognition.DetectModerationLabelsInput{
		Image: &rekognition.Image{
			Bytes: imageBytes,
		},
		MinConfidence: aws.Float64(90.0), // 设置置信度阈值
	}

	output, err := rekognitionSvc.DetectModerationLabels(input)
	if err != nil {
		return fmt.Errorf("调用 Rekognition 失败: %v", err)
	}

	// 检查图片内容
	if len(output.ModerationLabels) > 0 {
		log.Printf("[uploadImage] 检测到不适合的内容: %v", output.ModerationLabels)
		return fmt.Errorf("[uploadImage] 检测到不适合的内容: %v", output.ModerationLabels)
	}
	// for _, label := range output.ModerationLabels {
	// 	// 只检查特定类型的违规内容
	// 	switch *label.Name {
	// 	case "Porn", "Terrorism", "Politics", "Ad", "Bad":
	// 		if *label.Confidence > 90.0 {
	// 			return fmt.Errorf("图片包含违规内容: %s (置信度: %.2f%%)",
	// 				*label.Name, *label.Confidence)
	// 		}
	// 	}
	// }

	return nil
}

// 异步检查图片内容是否违规
func AsyncCheckImageContent(ctx context.Context, n *nucl.Nucleus, fileKey string) {
	// 创建一个新的 context，设置超时时间
	checkCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	svc := s3.New(n.AwsSession)
	err := CheckImageContent(checkCtx, n, svc, n.Config.Aws.AwsStorageBucketName, fileKey)
	if err != nil {
		sdlog.Errorf("[AsyncCheckImageContent] 图片内容违规: %v", err)
		// 删除违规图片
		_, err = svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(n.Config.Aws.AwsStorageBucketName),
			Key:    aws.String(fileKey),
		})
		if err != nil {
			sdlog.Errorf("[AsyncCheckImageContent] 删除违规图片失败: %v", err)
		}
		err = n.DB.WithContext(checkCtx).Model(&model.CheckInPointImage{}).
			Where("image_url = ?", fmt.Sprintf("%s/%s", n.Config.Aws.AwsS3CustomDomain, fileKey)).
			Update("status", common.UserImageStatusRejected).Error
		if err != nil {
			sdlog.Errorf("[AsyncCheckImageContent] 更新图片状态失败: %v", err)
		}
		sdlog.Infof("[AsyncCheckImageContent] 图片 %s 审核不通过", fileKey)
	} else {
		err = n.DB.WithContext(checkCtx).Model(&model.CheckInPointImage{}).
			Where("image_url = ?", fmt.Sprintf("%s/%s", n.Config.Aws.AwsS3CustomDomain, fileKey)).
			Update("status", common.UserImageStatusApproved).Error
		if err != nil {
			sdlog.Errorf("[AsyncCheckImageContent] 更新图片状态失败: %v", err)
		}
		sdlog.Infof("[AsyncCheckImageContent] 图片 %s 审核通过", fileKey)
	}
}

// 创建用户图片记录
func CreateUserImages(ctx context.Context, n *nucl.Nucleus, tx *gorm.DB, userID int64, fileKeys []string, spotID int64) error {
	// 1. 验证所有文件是否已上传（带重试）
	svc := s3.New(n.AwsSession)
	for _, fileKey := range fileKeys {
		if err := VerifyFileUpload(ctx, n, svc, n.Config.Aws.AwsStorageBucketName, fileKey); err != nil {
			sdlog.Errorf("[CreateUserImages] 验证文件失败: %v", err)
			return fmt.Errorf("验证文件失败: %v", err)
		}
	}

	// 2. 将所有图片信息存储到check_in_point_image表
	var spotImages []model.CheckInPointImage
	for i, fileKey := range fileKeys {
		spotImages = append(spotImages, model.CheckInPointImage{
			CheckInPointID: spotID,
			ImageURL:       fmt.Sprintf("%s/%s", n.Config.Aws.AwsS3CustomDomain, fileKey),
			SortIndex:      int32(i),
			Status:         common.SpotAuditStatusPending,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
	}

	if err := tx.WithContext(ctx).Create(&spotImages).Error; err != nil {
		sdlog.Errorf("[CreateUserImages] 保存图片信息失败: %v", err)
		return fmt.Errorf("保存图片信息失败: %v", err)
	}

	return nil
}

// 创建用户图片记录
func CreateProductImages(ctx context.Context, n *nucl.Nucleus, tx *gorm.DB, userID int64, fileKeys []string, productID int64) error {
	// 1. 验证所有文件是否已上传（带重试）
	// svc := s3.New(n.AwsSession)
	// for _, fileKey := range fileKeys {
	// 	if err := VerifyFileUpload(ctx, n, svc, n.Config.Aws.AwsStorageBucketName, fileKey); err != nil {
	// 		sdlog.Errorf("[CreateUserImages] 验证文件失败: %v", err)
	// 		return fmt.Errorf("验证文件失败: %v", err)
	// 	}
	// }
	if len(fileKeys) == 0 {
		return nil
	}
	// 2. 将所有图片信息存储到product_image表
	var productImages []model.ProductImage
	for i, fileKey := range fileKeys {
		productImages = append(productImages, model.ProductImage{
			ProductID: productID,
			ImageURL:  fmt.Sprintf("%s/%s", n.Config.Aws.AwsS3CustomDomain, fileKey),
			SortIndex: int32(i),
			Status:    common.SpotAuditStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ImageType: "large",
		})
	}

	if err := tx.WithContext(ctx).Create(&productImages).Error; err != nil {
		sdlog.Errorf("[CreateProductImages] 保存图片信息失败: %v", err)
		return fmt.Errorf("保存图片信息失败: %v", err)
	}

	return nil
}
