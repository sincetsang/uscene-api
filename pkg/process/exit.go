package process

import (
	"TMA/pkg/sdlog"
	"os"
	"os/signal"
	"syscall"
)

// GracefullyExit 优雅的退出，调用举例
//
//	go func() {
//		process.GracefullyExit(func() {
//			err := app.Shutdown(context.Background())
//			if err != nil {
//				sdlog.WithError(err).Error("shut down failed")
//			}
//		})
//	}()
func GracefullyExit(exitFunc func()) {
	osc := make(chan os.Signal, 1)
	signal.Notify(osc, syscall.SIGTERM, syscall.SIGINT)
	s := <-osc
	sdlog.Infof("exit signal, signal=%s", s)
	exitFunc()
}
