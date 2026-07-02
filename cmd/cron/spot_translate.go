package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/service"
	"TMA/service/db/model"
	"context"
	"fmt"
	"time"
)

const (
	spotTranslateLockKey = "spot_translate_lock"
	spotTranslateLockTTL = 10 * time.Minute
)

// translateSpotContent 翻译地标内容
func translateSpotContent(ctx context.Context, nu *nucl.Nucleus) error {
	// 尝试获取锁
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := redis.TryLock(ctx, nu, spotTranslateLockKey, lockValue, spotTranslateLockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !acquired {
		sdlog.Infof("Another instance is already running, exiting...")
		return nil
	}
	// 确保在函数结束时释放锁
	defer func() {
		if errDefer := redis.Unlock(ctx, nu, spotTranslateLockKey, lockValue); errDefer != nil {
			sdlog.Errorf("Failed to release lock: %v", errDefer)
		} else {
			sdlog.Infof("lock released")
		}
	}()

	// 查询需要翻译的地标
	var spots []model.CheckInPoint
	err = nu.DB.WithContext(ctx).
		Where("(title_cn = '' OR title_en = '' OR description_cn = '' OR description_en = '') AND status = 1 and audit_status = 1 and title != '' and description != ''").
		Find(&spots).Error
	if err != nil {
		return err
	}

	sdlog.Infof("找到 %d 个需要翻译的地标", len(spots))

	for _, spot := range spots {
		// 检查标题和描述是否需要翻译
		if spot.TitleCn == "" || spot.TitleEn == "" || spot.DescriptionCn == "" || spot.DescriptionEn == "" {
			// 使用 TranslateSpotContent 方法进行翻译
			service.TranslateSpotContent(nu.OpenaiApiKey, spot.ID, spot.Title, spot.Description)
			sdlog.Infof("[translateSpotContent] 地标翻译任务已提交: spot_id=%d", spot.ID)
		}
	}

	return nil
}
