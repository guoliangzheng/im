package service

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"fmt"

	"github.com/coreos/etcd/clientv3"
	"github.com/naoina/toml"
	"github.com/nats-io/nats"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

type Config struct {
	Common struct {
		Version  string
		IsDebug  bool
		LogLevel string
		LogPath  string
	}

	Apns struct {
		ClusterServers int
	}

	Etcd struct {
		Addrs     []string
		ApnsAddrs string
	}

	Nats struct {
		Addrs []string

		PubApnsTopic string
		PubApnsGroup string

		SetTokenTopic string
		SetTokenGroup string

		DelTokenTopic string
		DelTokenGroup string
	}
}

var Conf = &Config{}
var etcdCli *clientv3.Client

func loadConfig(staticConf bool) {
	var contents []byte
	var err error

	if staticConf {
		//静态配置
		contents, err = ioutil.ReadFile("configs/apns.toml")
	} else {
		contents, err = ioutil.ReadFile("/etc/gomqtt/apns.toml")
	}

	if err != nil {
		log.Fatal("load config error", zap.Error(err))
	}

	tbl, err := toml.Parse(contents)
	if err != nil {
		log.Fatal("parse config error", zap.Error(err))
	}

	toml.UnmarshalTable(tbl, Conf)

	fmt.Println(Conf)

	// 初始化Logger
	InitLogger(Conf.Common.LogPath, Conf.Common.LogLevel, Conf.Common.IsDebug)

	nc, err = initNatsConn()
	if err != nil {
		Logger.Fatal("init nats error", zap.Error(err))
	}

	// stream hot update
	etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   Conf.Etcd.Addrs,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		Logger.Fatal("connect to etcd error", zap.Error(err))
	}

	uploadEtcd(etcdCli)
}

func initNatsConn() (*nats.Conn, error) {
	opts := nats.DefaultOptions
	opts.Servers = Conf.Nats.Addrs
	opts.MaxReconnect = 1000
	opts.ReconnectWait = 5 * time.Second

	nc, err := opts.Connect()
	if err != nil {
		return nil, err
	}

	// Setup callbacks to be notified on disconnects and reconnects
	nc.Opts.DisconnectedCB = func(nc *nats.Conn) {
		Logger.Error("nats disconnected")
	}

	// See who we are connected to on reconnect.
	nc.Opts.ReconnectedCB = func(nc *nats.Conn) {
		Logger.Info("nats reconnected")
	}

	return nc, nil
}

func uploadEtcd(cli *clientv3.Client) {
	key := Conf.Etcd.ApnsAddrs + "/" + getHost()

	addr := talents.LocalIP()

	Logger.Debug("apns local ip", zap.String("ip", addr))

	go func() {
		for {
			// upload self ip
			Grant, err := etcdCli.Grant(context.TODO(), 15)
			if err != nil {
				Logger.Warn("etcd grant error", zap.Error(err))
				goto Sleep
			}

			_, err = etcdCli.Put(context.TODO(), key, addr, clientv3.WithLease(Grant.ID))
			if err != nil {
				Logger.Warn("etcd put error", zap.Error(err))
			}

		Sleep:
			time.Sleep(5 * time.Second)
		}
	}()

}

func getHost() string {
	host, err := os.Hostname()
	if err != nil {
		Logger.Fatal("get hostname error", zap.Error(err))
	}

	// in debug enviroment,we need to start several nodes in one machine,so pid is needed
	if Conf.Common.IsDebug {
		return fmt.Sprintf("%s-%d", host, os.Getpid())
	}

	return host
}
