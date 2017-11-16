package gate

/* mqtt connect包处理模块 */
import (
	"errors"
	"time"

	"im.zgl/gomqtt/global"
	proto "im.zgl/gomqtt/mqtt/protocol"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

func connect(ci *connInfo) error {
	reply := proto.NewConnackPacket()
	err, code := initConnection(ci)
	reply.SetReturnCode(code)
	if err != nil {
		Logger.Info("user connect failed", zap.Int64("cid", ci.id), zap.Error(err), zap.String("acc", talents.Bytes2String(ci.acc)),
			zap.String("user", talents.Bytes2String(ci.appID)))
		write(ci, reply)
		return err
	}

	if err := write(ci, reply); err != nil {
		Logger.Info("write connecaccept failed", zap.Int64("cid", ci.id), zap.Error(err),
			zap.String("acc", talents.Bytes2String(ci.acc)), zap.String("user", talents.Bytes2String(ci.appID)), zap.String("password", talents.Bytes2String(ci.cp.Password())))
		return err
	}

	return nil
}

func initConnection(ci *connInfo) (error, proto.ConnackCode) {
	// wait for connect and init connection
	// the first packet is connect type,we need to restrain the read deadline
	setReadDeadline(ci, time.Now().Add(10*time.Second))

	pt, err := read(ci)
	if err != nil {
		code, ok := err.(proto.ConnackCode)
		if ok {
			return errors.New("read connnect packet error"), code
		}
		return err, proto.ErrIdentifierRejected
	}

	cp, ok := pt.(*proto.ConnectPacket)
	if !ok {
		return errors.New("invalid packet type"), proto.ErrIdentifierRejected
	}

	ci.cp = cp

	// transfer account and user
	accTrans(ci)

	// validate the user
	ok = validate(ci.acc, ci.cp.Password())
	if !ok {
		return errors.New("validate failed"), proto.ErrIdentifierRejected
	}

	// if keepalive == 0 ,we should specify a default keepalive
	kp := ci.cp.KeepAlive()
	switch {
	case kp == 0:
		ci.cp.SetKeepAlive(Conf.Mqtt.DefaultKeepalive)

	case kp < Conf.Mqtt.MinKeepalive:
		ci.cp.SetKeepAlive(Conf.Mqtt.MinKeepalive)

	case kp > Conf.Mqtt.MaxKeepalive:
		ci.cp.SetKeepAlive(Conf.Mqtt.MaxKeepalive)

	default:
	}

	if ci.appID != nil {
		// 通过appid获取topics、compress、qos等信息
		// subscribe and login
		topics := make([][]byte, 1)
		qoses := make([]byte, 1)

		// Json协议下采用snappy压缩
		ci.compress = 211
		ci.payloadProtoType = global.PayloadText
		err = loginAndSub(ci, topics, qoses, 1)
		if err != nil {
			return err, proto.ErrServerUnavailable
		}
		ci.isInstantLogin = true
	}

	// Json协议下默认采用snappy压缩
	ci.compress = 200
	return nil, proto.ConnectionAccepted
}
