package uuid

import (
	"fmt"
	"time"

	"github.com/uber-go/zap"
)

// Standard ...
type Ts16 struct {
	In  chan int
	Out chan string
}

// Start ...
func (sd *Ts16) Start() {
	out := make(chan string)
	in := make(chan int)
	sd.Out = out
	sd.In = in

	go func() {
		defer func() {
			if err := recover(); err != nil {
				Logger.Warn("ts16 fatal error", zap.Error(err.(error)), zap.Stack())
			}
		}()

		clearTime := time.Now().Unix()
		var inc int64

		for {
			<-sd.In
			// 生成UUID
			uid := sd.gen(&clearTime, &inc)

			sd.Out <- uid
		}
	}()

	gen = sd
}

func (sd *Ts16) Close() {

}

// Gen ...
func (sd *Ts16) Gen() string {
	sd.In <- 1
	id := <-sd.Out

	return id
}

// 每一秒重新计数
func (sd *Ts16) gen(ct *int64, inc *int64) string {
	now := time.Now()
	if now.Unix() != *ct {
		*inc = 1
	} else {
		*inc++
	}
	*ct = now.Unix()

	var s string
	if *inc > 9999 {
		s = fmt.Sprintf("%d%02d%d", now.Unix(), sid, *inc)
	} else {
		s = fmt.Sprintf("%d%02d%04d", now.Unix(), sid, *inc)
	}

	return s
}
