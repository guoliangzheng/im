package gate

/* config配置管理,etcd管理等 */
import (
	"context"
	"errors"
	"io/ioutil"
	"time"

	"stathat.com/c/consistent"

	"fmt"

	"os"

	"sync"

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

	Gateway struct {
		WebDomain string
		ServerId  int
		NatsAddrs []string
	}

	Provider struct {
		Invoked   []string
		TcpAddr   string
		EnableTls bool
		TlsCert   string
		TlsKey    string

		WsAddr string
	}

	Etcd struct {
		Addrs   []string
		Streams string
		Rooms   string
	}

	Mqtt struct {
		QosMax byte

		DefaultKeepalive uint16
		MinKeepalive     uint16
		MaxKeepalive     uint16

		MaxUserLen     int
		MaxPasswordLen int
	}

	Dispatch struct {
		Addr string
	}

	// Mutex login
	Mutex struct {
		Type int
	}

	StreamAddrs map[string]string
	RoomAddrs   map[string]string
}

var Conf = &Config{}

// for dispatch
var Consist *consistent.Consistent

// for hash streams ip
var consist *consistent.Consistent

// for rpc to stream
var rpcRoutes = make(map[string]*rpcServie)
var mux = &sync.RWMutex{}

// nats.conn
var nc *nats.Conn

// 解析、加载、启动各类Config
func loadConfig(staticConf bool) error {
	var contents []byte
	var err error

	if staticConf {
		//静态配置
		contents, err = ioutil.ReadFile("configs/gateway.toml")
	} else {
		contents, err = ioutil.ReadFile("/etc/gomqtt/gateway.toml")
	}

	if err != nil {
		return fmt.Errorf("load config error:%v", err)
	}

	tbl, err := toml.Parse(contents)
	if err != nil {
		return fmt.Errorf("parse config error:%v", err)
	}

	err = toml.UnmarshalTable(tbl, Conf)
	if err != nil {
		return err
	}

	InitLogger(Conf.Common.LogPath, Conf.Common.LogLevel, Conf.Common.IsDebug)

	err = checkConfig()
	if err != nil {
		return err
	}

	nc, err = initNatsConn()
	if err != nil {
		return err
	}

	// stream hot update
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   Conf.Etcd.Addrs,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("can't connect to etcd:%v", err)
	}

	consist = consistent.New()
	Consist = consistent.New()

	watchEtcd(cli)
	uploadEtcd(cli)

	fmt.Println(Conf)

	return nil
}

// 检查config合法性
func checkConfig() error {
	if Conf.Mqtt.MinKeepalive < 10 {
		return errors.New("mqtt.minkeepalive mustn't below 10")
	}

	if Conf.Mqtt.DefaultKeepalive < 10 {
		return errors.New("mqtt.defaultkeepalive mustn't below 10")
	}

	if Conf.Mqtt.MaxKeepalive > 300 {
		return errors.New("mqtt.defaultkeepalive mustn't above 300")
	}

	return nil
}

// 通过Etcd实时获取stream的地址列表,并初始化对应的rpc连接
// update the stream addrs
//  etcdctl --endpoints="http://10.7.24.191:2379"  set "/gomqtt/gateway/dispatch/addr" :8906
// sudo confd -watch -backend etcd -node http://10.7.24.191:2379
func watchEtcd(cli *clientv3.Client) {
	// update the stream addrs
	go func() {
		Conf.StreamAddrs = make(map[string]string)
		rch := cli.Watch(context.TODO(), Conf.Etcd.Streams, clientv3.WithPrefix())

		for wresp := range rch {
			for _, ev := range wresp.Events {
				ip := string(ev.Kv.Value)
				if ev.Type == 0 { // PUT
					Conf.StreamAddrs[string(ev.Kv.Key)] = ip

					mux.Lock()
					if _, ok := rpcRoutes[ip]; !ok {
						rpc := &rpcServie{}
						if err := rpc.init(ip); err != nil {
							Logger.Info("rpc init error", zap.Error(err), zap.String("ip", ip))
							continue
						}
						rpcRoutes[ip] = rpc
					}
					mux.Unlock()
				} else {
					ip, ok := Conf.StreamAddrs[string(ev.Kv.Key)]
					if ok {
						mux.Lock()
						rpc, ok := rpcRoutes[ip]
						if ok {
							rpc.close()
						}
						delete(rpcRoutes, ip)
						mux.Unlock()
					}

					delete(Conf.StreamAddrs, string(ev.Kv.Key))
				}
			}

			consist = consistent.New()
			for _, v := range Conf.StreamAddrs {
				consist.Add(v)
			}

			// Logger.Debug("get new stream addrs", zap.Object("addrs", Conf.StreamAddrs))
		}
	}()

	// update the room addrs
	go func() {
		Conf.RoomAddrs = make(map[string]string)
		rch := cli.Watch(context.TODO(), Conf.Etcd.Rooms, clientv3.WithPrefix())

		for wresp := range rch {
			for _, ev := range wresp.Events {
				v := string(ev.Kv.Value)
				if ev.Type == 0 { // PUT
					Conf.RoomAddrs[string(ev.Kv.Key)] = v
				} else {
					delete(Conf.RoomAddrs, string(ev.Kv.Key))
				}
			}

			Consist = consistent.New()
			for _, v := range Conf.RoomAddrs {
				Consist.Add(v)
			}

			// Logger.Debug("get new room addrs", zap.Object("addrs", Conf.RoomAddrs), zap.Object("consist_addrs", Consist.Members()))
		}
	}()
}

// 将本gateway的ip或者域名上传到etcd，以供dispatch服务使用
func uploadEtcd(cli *clientv3.Client) {
	key := Conf.Etcd.Rooms + "/" + getHost()

	var addr string
	if Conf.Gateway.WebDomain == "" {
		addr = talents.LocalIP()
	} else {
		addr = Conf.Gateway.WebDomain
	}

	Logger.Debug("local ip", zap.String("ip", addr))

	go func() {
		for {
			// upload self ip
			Grant, err := cli.Grant(context.TODO(), 30)
			if err != nil {
				Logger.Warn("etcd grant error", zap.Error(err))
				goto Sleep
			}

			_, err = cli.Put(context.TODO(), key, addr, clientv3.WithLease(Grant.ID))
			if err != nil {
				Logger.Warn("etcd put error", zap.Error(err))
			}

		Sleep:
			time.Sleep(10 * time.Second)
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

func initNatsConn() (*nats.Conn, error) {
	opts := nats.DefaultOptions
	opts.Servers = Conf.Gateway.NatsAddrs
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
