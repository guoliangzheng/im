package gate

/* Mqtt协议Subscribe报文处理模块 */
import (
	"errors"

	proto "im.zgl/gomqtt/mqtt/protocol"

	"fmt"

	rpc "im.zgl/gomqtt/proto"
)

func subscribe(ci *connInfo, p *proto.SubscribePacket) error {
	topics, rets, err := topicsAndRets(ci, p.Topics(), p.Qos())
	if err != nil {
		return err
	}

	rpcH, err := getRpc(ci)
	if err != nil {
		return err
	}

	err = rpcH.subscribe(&rpc.SubMsg{
		Acc:   ci.acc,
		AppID: ci.appID,
		Ts:    topics,
	})
	if err != nil {
		return fmt.Errorf("subscribe rpc error: %v", err)
	}

	// give back the suback
	pb := proto.NewSubackPacket()
	pb.SetPacketID(p.PacketID())

	// return the final qos level
	pb.AddReturnCodes(rets)
	write(ci, pb)

	return nil
}

func unsubscribe(ci *connInfo, p *proto.UnsubscribePacket) error {
	topics, err := topics(p)
	if err != nil {
		return err
	}

	rpcH, err := getRpc(ci)
	if err != nil {
		return err
	}

	err = rpcH.unSubscribe(&rpc.UnSubMsg{
		Acc:   ci.acc,
		AppID: ci.appID,
		Ts:    topics,
	})
	if err != nil {
		return fmt.Errorf("unSubscribe error: %v", err)
	}

	pb := proto.NewUnsubackPacket()
	pb.SetPacketID(p.PacketID())

	write(ci, pb)
	return nil
}

func topicsAndRets(ci *connInfo, tps [][]byte, qoses []byte) ([]*rpc.Topic, []byte, error) {
	rets := make([]byte, 0, len(tps))
	topics := make([]*rpc.Topic, 0, len(tps))

	for i, t := range tps {
		tp, ty, err := topicTrans(t)
		if err != nil {
			return nil, nil, err
		}

		// set master topic
		if ty == 1000 {
			// get appid and payload proto type
			err = appidTrans(ci, tp)
			if err != nil {
				return nil, nil, err
			}
		}

		qos := qosTrans(qoses[i])

		topic := &rpc.Topic{
			Qos: int32(qos),
			Tp:  tp,
			Ty:  int32(ty),
		}

		topics = append(topics, topic)
		rets = append(rets, qos)
	}

	// 主topic必须在第一次订阅提供，因此这里需要验证主topic是否存在
	if ci.appID == nil {
		return nil, nil, errors.New("need provide master topic")
	}

	return topics, rets, nil
}

func topics(p *proto.UnsubscribePacket) ([]*rpc.Topic, error) {
	topics := make([]*rpc.Topic, 0, len(p.Topics()))

	for _, t := range p.Topics() {
		tp, ty, err := topicTrans(t)
		if err != nil {
			return nil, err
		}

		// forbid to unsubsribe master topic
		if ty == 1000 {
			return nil, errors.New("forbid to unsubsribe master topic")
		}
		topic := &rpc.Topic{
			Tp: tp,
		}

		topics = append(topics, topic)
	}

	return topics, nil
}
