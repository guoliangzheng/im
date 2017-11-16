package service

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/nats-io/nats"
	"github.com/uber-go/zap"
)

type Apns struct {
	Stopped bool
}

func New() *Apns {
	return &Apns{}
}

// nats connection
var nc *nats.Conn

func (g *Apns) Start(isStatic bool) {
	loadConfig(isStatic)

	// tempory disable reload
	// go adminStart()

	// 启动等待，所有apns节点启动完毕才能继续运行
	for {
		if g.Stopped {
			break
		}

		resp, err := etcdCli.Get(context.Background(), Conf.Etcd.ApnsAddrs, clientv3.WithPrefix())
		if err != nil {
			Logger.Info("etcd get error", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}

		if int(resp.Count) != Conf.Apns.ClusterServers {
			Logger.Info("etcd get count error", zap.Int64("count", resp.Count))
			time.Sleep(2 * time.Second)
			continue
		}

		break
	}

	Logger.Info("apns started ok")
}
