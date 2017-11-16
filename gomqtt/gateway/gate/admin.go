package gate

/* 管理和监控服务 */
import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/uber-go/zap"
)

func adminStart() {
	e := echo.New()

	// configuration hot update
	e.GET("/reload", reload)

	err := e.Start(":8907")
	if err != nil {
		Logger.Fatal("echo http start failed", zap.Error(err))
	}
}

func reload(c echo.Context) error {
	loadConfig(false)

	c.Response().Writer()
	return nil
}

func monitorLeaking() {
	for {
		r := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -n |awk '{print $2}'|sort|uniq -c | grep %v", os.Getpid()))
		v, _ := r.Output()
		fds := strings.Split(string(v), " ")[2]
		Logger.Debug("goroutine和fd数目", zap.Int("goroutine", runtime.NumGoroutine()), zap.String("fd", fds))

		time.Sleep(20 * time.Second)

	}

}

func monitorsStart() {
	// monitor the goroutine and file descriptor leaking
	go monitorLeaking()
}
