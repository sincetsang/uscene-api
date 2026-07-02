package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
)

// 预签名URL请求参数
type PresignedURLRequest []string // 文件名列表（必须包含扩展名）

// 预签名URL响应
type PresignedURLResponse []FileUploadInfo // 文件上传信息列表

type FileUploadInfo struct {
	UploadURL string `json:"upload_url"` // 上传URL
	FileURL   string `json:"file_url"`   // 文件访问URL
	FileKey   string `json:"file_key"`   // 文件在S3中的key
}

// 生成唯一文件名
func generateUniqueFileName(userId int64, ext string) string {
	// 使用纳秒级时间戳和随机数确保唯一性
	timestamp := time.Now().UnixNano()
	random := rand.Int63n(1000) // 0-999的随机数
	return fmt.Sprintf("user/%d_%d_%d%s", userId, timestamp, random, ext)
}

// 获取文件的MIME类型
func getMimeType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "image/" + strings.TrimPrefix(ext, ".")
	}
}

// 获取预签名URL
func getPresignedURL(ec *middleware.AppRequestContext) error {
	// 1. 检查用户今日上传图片数量
	userId := ec.AuthData.User.ID
	today := time.Now().Truncate(24 * time.Hour)
	var count int64
	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserImage{}).
		Where("user_id = ? AND created_at >= ?", userId, today).
		Count(&count).Error
	if err != nil {
		sdlog.Errorf("[getPresignedURL] 查询用户图片数量失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 2. 解析请求参数
	var req PresignedURLRequest
	if err = ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查文件数量是否超过限制
	if count+int64(len(req)) > 200 {
		sdlog.Errorf("[getPresignedURL] 用户今日上传图片数量将超过上限: 当前%d, 请求%d", count, len(req))
		return webapi.Error(common.ErrUploadLimit).Render(ec)
	}

	// 3. 处理每个文件
	var fileUploadInfos []FileUploadInfo
	for _, fileName := range req {
		// 验证文件类型
		ext := strings.ToLower(filepath.Ext(fileName))
		allowedExts := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
		}
		if !allowedExts[ext] {
			sdlog.Errorf("[getPresignedURL] 不支持的文件类型: %s", ext)
			return webapi.Error(common.ErrUnsupportedFileType).Render(ec)
		}

		// 生成唯一文件名
		fileKey := generateUniqueFileName(ec.AuthData.User.ID, ext)

		// 生成预签名URL
		svc := s3.New(ec.Nu.AwsSession)
		putObjectInput := &s3.PutObjectInput{
			Bucket:      aws.String(ec.Nu.Config.Aws.AwsStorageBucketName),
			Key:         aws.String(fileKey),
			ContentType: aws.String(getMimeType(ext)),
		}

		// 创建预签名URL请求
		presignReq, _ := svc.PutObjectRequest(putObjectInput)

		// 添加条件限制
		presignReq.Handlers.Sign.PushBack(func(r *request.Request) {
			// 设置文件大小限制为2MB
			r.HTTPRequest.Header.Set("x-amz-content-length-range", "0,2097152")
			// 设置内容类型限制
			r.HTTPRequest.Header.Set("x-amz-content-type", getMimeType(ext))
			// 设置缓存控制
			r.HTTPRequest.Header.Set("Cache-Control", "max-age=31536000")
		})

		// 设置预签名URL的有效期（例如5分钟）
		url, err := presignReq.Presign(24 * time.Hour)
		if err != nil {
			sdlog.Errorf("[getPresignedURL] 生成预签名URL失败: %v", err)
			return webapi.Error(common.ErrService).Render(ec)
		}

		// 生成文件访问URL
		fileURL := fmt.Sprintf("%s/%s", ec.Nu.Config.Aws.AwsS3CustomDomain, fileKey)

		fileUploadInfos = append(fileUploadInfos, FileUploadInfo{
			UploadURL: url,
			FileURL:   fileURL,
			FileKey:   fileKey,
		})
	}

	// 4. 返回所有文件的预签名URL和访问URL
	return webapi.OK(PresignedURLResponse(fileUploadInfos)).Render(ec)
}
