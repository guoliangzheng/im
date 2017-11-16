package gate

/* 接收从Stream发送到客户端的消息，Pub2Client */
import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats"
	"im.zgl/gomqtt/global"
	"github.com/uber-go/zap"

	proto "im.zgl/gomqtt/mqtt/protocol"

	rpc "im.zgl/gomqtt/proto"
	"im.zgl/talents"
)

// 从nats接收订阅消息，然后推送给客户端
func pub2c(ci *connInfo, msg *nats.Msg) {
	//如果还未登录成功，等待500毫秒
	//这里解决stream登录和订阅消息的异步问题: login rpc还没返回，但是订阅的消息已经发送古来
	if !ci.isSubed {
		time.Sleep(500 * time.Millisecond)
	}

	switch ci.payloadProtoType {
	case global.PayloadJson:
		d := &global.JsonMsgs{}
		_, err := d.UnmarshalMsg(msg.Data)
		if err != nil {
			Logger.Info("unmarshal JsonMsgs error", zap.Error(err), zap.Int64("cid", ci.id))
			return
		}

		err = pubJson(ci, d)
		if err != nil {
			Logger.Debug("pubJson error", zap.Error(err))
		}

	case global.PayloadText:
		d := &global.TextMsgs{}
		_, err := d.UnmarshalMsg(msg.Data)
		if err != nil {
			Logger.Info("unmarshal TextMsgs error", zap.Error(err), zap.Int64("cid", ci.id))
			return
		}

		err = pubText(ci, d)
		if err != nil {
			Logger.Debug("pubText error", zap.Error(err))
		}
	case global.PayloadProtobuf:

	}

}

// publish Text格式的消息
func pubText(ci *connInfo, m *global.TextMsgs) error {
	for _, msg := range m.Msgs {
		p := proto.NewPublishPacket()
		p.SetQoS(byte(msg.Qos))
		//@Optimize
		topic := bytes.Join([][]byte{msg.FTopic, msg.FAcc}, topicSep)
		p.SetTopic(topic)
		p.SetPayload(msg.Msg)

		id, err := mapTextID(ci, msg.FTopic, msg.MsgID, msg.Qos)
		if err != nil {
			return err
		}

		p.SetPacketID(id)

		Logger.Debug("recv nats msg", zap.String("msg", string(msg.Msg)), zap.Int("id", int(id)))
		err = write(ci, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func pubJson(ci *connInfo, msg *global.JsonMsgs) error {
	p := proto.NewPublishPacket()
	p.SetQoS(byte(msg.Qos))

	j := &global.Messages{}

	j.Compress = ci.compress

	// 对Msgs列表进行JSON编码
	d, err := msg.Data.MarshalJSON()
	if err != nil {
		return nil
	}

	// 对Json编码后的数据进行压缩
	data, err := compress(ci, d)
	if err != nil {
		return err
	}
	j.Data = talents.Bytes2String(data)

	b, err := j.MarshalJSON()
	if err != nil {
		return err
	}

	p.SetPayload(b)
	id, err := mapID(ci, msg.TTopics, msg.MsgID, msg.Qos)
	p.SetPacketID(id)
	p.SetTopic(ci.appID)

	Logger.Debug("recv nats msg", zap.Int("id", int(id)))
	err = write(ci, p)
	if err != nil {
		return err
	}

	return nil
}

func mapTextID(ci *connInfo, topic []byte, mid []byte, qos int32) (uint16, error) {
	id, err := getID(ci)
	if err != nil {
		return 0, err
	}

	// 只有在qos不为0时，才保存id映射，为后面ack的删除做备用
	if qos != 0 {
		acks := make([]*rpc.AckTopicMsgID, 1)
		acks[0] = &rpc.AckTopicMsgID{
			Tp:  topic,
			Mid: mid,
		}

		ci.rwm.Lock()
		ci.idMap[id] = acks
		ci.rwm.Unlock()
	}

	return id, nil
}

func mapID(ci *connInfo, topics [][]byte, ids [][]byte, qos int32) (uint16, error) {
	if len(topics) != len(ids) {
		return 0, errors.New("topics and msgIDs must have the same length")
	}

	id, err := getID(ci)
	if err != nil {
		return 0, err
	}

	// 只有在qos不为0时，才保存id映射，为后面ack的删除做备用
	if qos != 0 {
		acks := make([]*rpc.AckTopicMsgID, len(ids))
		for k, v := range ids {
			acks[k] = &rpc.AckTopicMsgID{
				Tp:  topics[k],
				Mid: v,
			}
		}

		ci.idMap[id] = acks
	}

	return id, nil
}

// 自加MsgID,超过65535返回错误
func getID(ci *connInfo) (uint16, error) {
	if ci.msgID >= 65535 {
		return 0, fmt.Errorf("MsgID Beyond the maximum range")
	}
	ci.msgID++
	return ci.msgID, nil
}
