package nucl

import (
	"TMA/pkg/sderr"
	"TMA/pkg/sdgorm"
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	globalDB *gorm.DB
	dbOnce   sync.Once
)

// GetGlobalDB 获取全局数据库连接
func GetGlobalDB() *gorm.DB {
	return globalDB
}

func (nu *Nucleus) initDB() error {
	if !nu.Options.DB {
		return nil
	}
	return nu.withLog("DB", func() error {
		mainAddr := nu.Config.DB
		if nu.Options.GormLogger != "" {
			mainAddr.Logger = nu.Options.GormLogger
		}
		gConfig := &gorm.Config{}
		if nu.Env.IsDevelop() {
			gConfig = &gorm.Config{
				Logger: logger.New(
					log.New(os.Stdout, "\r\n", log.LstdFlags), // 输出到标准输出
					logger.Config{
						SlowThreshold:             time.Second, // 慢 SQL 阈值
						LogLevel:                  logger.Info, // 日志级别
						IgnoreRecordNotFoundError: false,       // 忽略ErrRecordNotFound（记录未找到）错误
						Colorful:                  false,       // 禁用彩色打印
					},
				),
			}
		}
		db, err := sdgorm.Dial(mainAddr, gConfig)
		if err != nil {
			return sderr.WithStack(err)
		}

		nu.DB = db
		// 初始化全局数据库连接
		dbOnce.Do(func() {
			globalDB = db
		})
		return nil
	})
}

type DataOp func(tx *gorm.DB, arg interface{}) error

func (nu *Nucleus) ApplyData(ctx context.Context, arg interface{}, funcs ...DataOp) error {
	if len(funcs) <= 0 {
		return nil
	}
	err := nu.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, f := range funcs {
			if f != nil {
				if err0 := f(tx, arg); err0 != nil {
					return sderr.WithStack(err0)
				}
			}
		}
		return nil
	})
	return sderr.WithStack(err)
}

func (nu *Nucleus) MustApplyDataForTest(ctx context.Context, t *testing.T, arg interface{}, funcs ...DataOp) {
	err := nu.ApplyData(ctx, arg, funcs...)
	if err != nil {
		t.Fatal(err)
		return
	}
}
