package gate

/* 进入gateway的长链接服务主goroutine
provider ---->   serve */

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	proto "im.zgl/gomqtt/mqtt/protocol"

	"im.zgl/talents"
	"github.com/uber-go/zap"

	"github.com/nats-io/nats"
	rpc "im.zgl/gomqtt/proto"
	"im.zgl/gomqtt/uuid"

	"im.zgl/gomqtt/mqtt/service"
)

type connInfo struct {
	// 连接id
	id int64

	// conn Type: 1.tcp 2.websocket 3.http
	tp int
	// tcp conn
	c net.Conn
	// websocket conn
	wsC *websocket.Conn

	cp *proto.ConnectPacket

	rip string

	// 入包数量
	inCount int
	// 出包数量
	outCount int

	relogin chan struct{}

	test []byte

	acc []byte

	// appID只能是以下几种形式：
	// 1.username--appid传递的，这里的appid是在服务器做了Topics管理的(动态Topics管理)
	// 2.通过主topic type == 1000 来传递的，这里是静态类型的appid.在这种情况下，connect时首先将
	// appid设置为ClientID，然后后续subscribe时，再替换为主topic，若没有主topic，那么
	// 当前连接时异常的，必须断开
	appID []byte

	// 是否订阅和login
	isSubed bool
	// 是否立即登录，如果通过appid来管理topics的，那么就是立即登录
	// gateway会通过appid去自动获取预先设定的所有topics，并进行订阅
	// 非立即登录：用户在主动订阅时，才进行登录
	isInstantLogin bool

	// 是否已订阅nats
	isNatsSubed bool
	// nats订阅的handler，用于取消订阅
	natsHandler *nats.Subscription

	// payload使用的协议类型
	// 可选3种： PlayText、Protobuf、Json
	payloadProtoType int32

	// Json格式的Payload才能进行压缩
	// 压缩信息 百位保存压缩算法，十位压缩级别，个位保存是否压缩
	// 百位，1:gzip,2:snappy
	// 十位
	// 个位: 1压缩，其它不压缩
	compress int

	// 当前的mqtt id
	// 该id是mqtt协议使用的uint16 id,通过真实的msgid映射而来
	msgID uint16
	// 保存id映射关系表
	idMap map[uint16][]*rpc.AckTopicMsgID

	// 互斥锁
	rwm *sync.RWMutex
}

func serve(ci *connInfo) {
	defer func() {
		if err := recover(); err != nil {
			Logger.Info("user's main goroutine has a panic error", zap.Error(err.(error)), zap.Stack())
		}
	}()

	//generate a uuid for this conn
	ci.id = uuid.Gen()

	// set rip
	setIP(ci)

	Logger.Debug("a new connection has established", zap.Int64("cid", ci.id), zap.String("ip", ci.rip))

	defer func() {
		closeConn(ci)

		if ci.isNatsSubed {
			err := ci.natsHandler.Unsubscribe()
			if err != nil {
				Logger.Info("unsubscribe error", zap.Error(err), zap.Int64("cid", ci.id))
			}
		}

		if ci.isSubed {
			delMutex(ci)
		}

		close(ci.relogin)
	}()

	ci.relogin = make(chan struct{})
	ci.idMap = make(map[uint16][]*rpc.AckTopicMsgID)
	ci.rwm = &sync.RWMutex{}
	//----------------Connection init---------------------------------------------
	err := connect(ci)
	if err != nil {
		return
	}

	Logger.Debug("user connected ok!", zap.String("acc", talents.Bytes2String(ci.acc)),
		zap.String("user", talents.Bytes2String(ci.appID)), zap.String("password", talents.Bytes2String(ci.cp.Password())), zap.Int64("cid", ci.id), zap.Float64("keepalive", float64(ci.cp.KeepAlive())))

	wait := time.Duration(ci.cp.KeepAlive()+10) * time.Second

	for {
		if !ci.isSubed {
			// if not subscribed，only wait for 10 second
			setReadDeadline(ci, time.Now().Add(time.Duration(Conf.Mqtt.MinKeepalive-5)*time.Second))
		} else {
			setReadDeadline(ci, time.Now().Add(wait))
		}

		// We need to considering about the network delay,so here allows 10 seconds delay.
		pt, err := read(ci)
		if err != nil {
			goto STOP
		}

		err = processPacket(ci, pt)
		if err != nil {
			Logger.Info("process packet error", zap.Error(err), zap.Int64("cid", ci.id))
			goto STOP
		}

	}

STOP:
	if ci.isSubed {
		rpcH, err := getRpc(ci)
		if err != nil {
			Logger.Info("logout get rpc error", zap.Error(err))
			return
		}

		err = rpcH.logout(&rpc.LogoutMsg{
			Acc:   ci.acc,
			AppID: ci.appID,
		})

		Logger.Debug("user logout", zap.Error(err), zap.Int64("cid", ci.id))
	}
}

func setReadDeadline(ci *connInfo, t time.Time) {
	switch ci.tp {
	case 1:
		ci.c.SetReadDeadline(t)
	case 2:
		ci.wsC.SetReadDeadline(t)
	}
}

func setIP(ci *connInfo) {
	switch ci.tp {
	case 1:
		ci.rip = ci.c.RemoteAddr().String()
	case 2:
		ci.rip = ci.wsC.RemoteAddr().String()
	}
}

func closeConn(ci *connInfo) {
	switch ci.tp {
	case 1:
		ci.c.Close()
	case 2:
		ci.wsC.Close()
	}
}

func read(ci *connInfo) (proto.Packet, error) {
	var pt proto.Packet
	var buf []byte
	var n int
	var err error

	switch ci.tp {
	case 1:
		pt, buf, n, err = service.ReadPacket(ci.c)
		if err != nil {
			nerr, ok := err.(net.Error)
			if ok && nerr.Timeout() {
				Logger.Debug("user connect but not subscribed, disconnected", zap.Int64("cid", ci.id))
			} else {
				Logger.Warn("Read packet error", zap.Error(err), zap.String("buf", fmt.Sprintf("%v", buf)), zap.Int("bytes", n), zap.Int64("cid", ci.id))
			}

			return nil, err
		}

	case 2:
		pt, err = service.ReadWsPacket(ci.wsC)
		if err != nil {
			nerr, ok := err.(net.Error)
			if ok && nerr.Timeout() {
				Logger.Debug("user connect but not subscribed, disconnected", zap.Int64("cid", ci.id))
			} else {
				Logger.Warn("Read packet error", zap.Error(err), zap.Int64("cid", ci.id))
			}

			return nil, err
		}
	}

	return pt, nil
}

func write(ci *connInfo, p proto.Packet) error {
	var err error
	switch ci.tp {
	case 1:
		err = service.WritePacket(ci.c, p)
	case 2:
		err = service.WriteWsPacket(ci.wsC, p)
	}

	return err
}
