package gate

/* Mqtt协议Publish报文处理模块 */
import (
	"errors"

	"im.zgl/gomqtt/global"
	proto "im.zgl/gomqtt/mqtt/protocol"
	"im.zgl/gomqtt/uuid"
	"im.zgl/talents"
	"github.com/uber-go/zap"

	"bytes"

	"fmt"

	rpc "im.zgl/gomqtt/proto"
)

//@ToDo
//从客户端过来的只能是私聊或者单播
func publish(ci *connInfo, p *proto.PublishPacket) error {
	ci.inCount++

	switch ci.payloadProtoType {
	case global.PayloadJson:
		c2s := &global.C2SMsg{}
		err := c2s.UnmarshalJSON(p.Payload())
		if err != nil {
			return fmt.Errorf("unmarshal error: %v, data: %s", err, p.Payload())
		}

		var mid string
		if c2s.MsgID == "" {
			mid = uuid.GenStr()
		} else {
			mid = c2s.MsgID
		}

		Logger.Debug("client publish JSON", zap.String("mid", mid), zap.String("topic", talents.Bytes2String(p.Topic())),
			zap.String("acc", c2s.ToAcc), zap.Int("in_count", ci.inCount))

		rpcH, err := getRpc(ci)
		if err != nil {
			return err
		}
		err = rpcH.pubJson(&rpc.PubJsonMsg{
			FAcc:    ci.acc,
			Ftp:     ci.appID,
			ToAcc:   talents.String2Bytes(c2s.ToAcc),
			Ttp:     p.Topic(),
			Qos:     int32(p.QoS()),
			Mid:     talents.String2Bytes(mid),
			Msg:     talents.String2Bytes(c2s.Msg),
			MsgType: int32(c2s.Type),
		})
		if err != nil {
			return fmt.Errorf("pubJson rpc error: %v", err)
		}

	case global.PayloadText:
		// text格式，需要生成MsgID
		mid := talents.String2Bytes(uuid.GenStr())
		tps := bytes.Split(p.Topic(), topicSep)
		if len(tps) != 2 {
			return errors.New("invalid publish topic, text topic need to be topic--acc")
		}

		qos := qosTrans(p.QoS())

		Logger.Debug("client publish text", zap.String("topic", string(tps[0])),
			zap.String("acc", string(tps[1])), zap.Int("in_count", ci.inCount))

		rpcH, err := getRpc(ci)
		if err != nil {
			return err
		}
		err = rpcH.pubText(&rpc.PubTextMsg{
			FAcc:  ci.acc,
			Ftp:   ci.appID,
			ToAcc: tps[1],
			Ttp:   tps[0],
			Qos:   int32(qos),
			Mid:   mid,
			Msg:   p.Payload(),
		})
		if err != nil {
			return fmt.Errorf("pubText rpc error: %v", err)
		}

	case global.PayloadProtobuf:

	}

	// need give back the ack
	if p.QoS() >= 1 {
		pb := proto.NewPubackPacket()
		pb.SetPacketID(p.PacketID())
		write(ci, pb)
	}

	return nil
}

// 将消息ID反映射后，投递到stream删除
func puback(ci *connInfo, p *proto.PubackPacket) error {
	mid := p.PacketID()

	ci.rwm.RLock()
	ids, ok := ci.idMap[mid]
	ci.rwm.RUnlock()
	if ok {
		rpcH, err := getRpc(ci)
		if err != nil {
			return err
		}
		err = rpcH.puback(&rpc.PubAckMsg{
			Acc:  ci.acc,
			Plty: ci.payloadProtoType,
			Mids: ids,
		})
		if err != nil {
			return fmt.Errorf("puback rpc error : %v", err)
		}

		ci.rwm.Lock()
		delete(ci.idMap, mid)
		ci.rwm.Unlock()
	}

	return nil
}
