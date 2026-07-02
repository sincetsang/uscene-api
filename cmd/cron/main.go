package main

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	sdrand.InitSeed()
	app := &cli.App{
		Name:  "cron",
		Usage: "定时任务",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   os.Getenv("TMA_CONFIG"),
				Usage:   "配置文件路径",
			},
		},
		Action: main0,
	}
	err := app.Run(os.Args)
	if err != nil {
		sdlog.Fatal(err)
	}
}

func main0(cc *cli.Context) error {
	time.Local = time.UTC
	configLoc := cc.String("config")

	config, err := nucl.LoadConfig(configLoc, common.ServiceNameCron)
	if err != nil {
		return sderr.WithStack(err)
	}

	nu, err := nucl.NewByConfig(*config, nucl.Options{
		DB:          true,
		ServiceName: common.ServiceNameCron,
		Redis:       true,
		Aws:         true,
		Snowflake:   true,
	})
	if err != nil {
		return sderr.WithStack(err)
	}
	defer nu.Close()

	// 创建 cron 调度器，使用秒级精度
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))

	ctx := context.Background()

	// 注册 instant_examine 任务：每10秒钟
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "spotToExamine", spotToExamine)
	})
	if err != nil {
		return fmt.Errorf("注册 spotToExamine 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: spotToExamine (每10秒钟执行一次)")

	// 注册 pay_timeout 任务：每5秒执行一次
	_, err = c.AddFunc("*/5 * * * * *", func() {
		runTask(ctx, nu, "payTimeoutCheck", payTimeoutCheck)
	})
	if err != nil {
		return fmt.Errorf("注册 payTimeoutCheck 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: payTimeoutCheck (每5秒执行一次)")

	// 注册 product_examine 任务：每10秒钟执行一次
	_, err = c.AddFunc("*/5 * * * * *", func() {
		runTask(ctx, nu, "productToExamine", productToExamine)
	})
	if err != nil {
		return fmt.Errorf("注册 productToExamine 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: productToExamine (每10秒钟执行一次)")

	// 注册 product_order_update 任务：每10秒钟执行一次
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "productOrderUpdate", productOrderUpdate)
	})
	if err != nil {
		return fmt.Errorf("注册 productOrderUpdate 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: productOrderUpdate (每10秒钟执行一次)")

	// 注册 checkin_examine 任务：每10秒钟执行一次
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "checkinToExamine", checkinToExamine)
	})
	if err != nil {
		return fmt.Errorf("注册 checkinToExamine 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: checkinToExamine (每10秒钟执行一次)")

	// 注册 member_order_update 任务：每10秒钟执行一次
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "memberOrderUpdate", memberOrderUpdate)
	})
	if err != nil {
		return fmt.Errorf("注册 memberOrderUpdate 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: memberOrderUpdate (每10秒钟执行一次)")

	// 注册 verification_order_update 任务：每10秒钟执行一次
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "verificationOrderUpdate", verificationOrderUpdate)
	})
	if err != nil {
		return fmt.Errorf("注册 verificationOrderUpdate 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: verificationOrderUpdate (每10秒钟执行一次)")

	// 注册 translateSpotContent 任务：每5秒钟执行一次
	_, err = c.AddFunc("*/10 * * * * *", func() {
		runTask(ctx, nu, "translateSpotContent", translateSpotContent)
	})
	if err != nil {
		return fmt.Errorf("注册 translateSpotContent 任务失败: %v", err)
	}
	sdlog.Info("已注册定时任务: translateSpotContent (每10秒钟执行一次)")

	// 启动 cron 调度器
	c.Start()
	sdlog.Info("定时任务调度器已启动")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	sdlog.Info("收到停止信号，正在关闭定时任务调度器...")
	c.Stop()

	return nil
}

// runTask 执行定时任务的通用包装函数
func runTask(ctx context.Context, nu *nucl.Nucleus, taskName string, taskFunc func(context.Context, *nucl.Nucleus) error) {
	startTime := time.Now()
	sdlog.WithField("job", taskName).Info("start cron job")
	defer func() {
		duration := time.Since(startTime)
		sdlog.WithField("duration", duration).WithField("job", taskName).Info("end cron job")
	}()

	err := taskFunc(ctx, nu)
	if err != nil {
		sdlog.WithField("job", taskName).Errorf("定时任务执行失败: %v", err)
	}
}
