package gate

/* 处理mqtt协议包逻辑的主体模块,Mqtt包解析在serve模块 */
import (
	"errors"

	"fmt"

	proto "im.zgl/gomqtt/mqtt/protocol"
)

func processPacket(ci *connInfo, pt proto.Packet) error {
	var err error

	switch p := pt.(type) {
	case *proto.DisconnectPacket: // recv Disconnect
		err = errors.New("recv disconnect packet")

	case *proto.PublishPacket: // recv publish
		if !ci.isSubed {
			return errors.New("publish need sub and login")
		}

		err = publish(ci, p)
	case *proto.PubackPacket:
		err = puback(ci, p)

	case *proto.SubscribePacket:
		// 如果是通过appid管理的topic订阅方案，那么不允许再主动订阅
		if ci.isInstantLogin {
			err = errors.New("you cant subscribe any topics after instant login")
		} else {
			// 还未订阅，进行登录和订阅
			if !ci.isSubed {
				err = loginAndSub(ci, p.Topics(), p.Qos(), p.PacketID())
			} else {
				err = subscribe(ci, p)
			}
		}

	case *proto.UnsubscribePacket:
		err = unsubscribe(ci, p)

	case *proto.PingreqPacket:
		pingReq(ci)

	default:
		err = fmt.Errorf("recv invalid packet type: %T", pt)
	}

	return err
}

func pingReq(ci *connInfo) {
	pb := proto.NewPingrespPacket()
	write(ci, pb)
}
