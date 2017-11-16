package gate

/* 登录一体化模块 */
import (
	"strconv"

	"github.com/nats-io/nats"
	"im.zgl/talents"

	"fmt"

	proto "im.zgl/gomqtt/mqtt/protocol"
	rpc "im.zgl/gomqtt/proto"
)

// login and first subscribe
func loginAndSub(ci *connInfo, tps [][]byte, qoses []byte, pid uint16) error {
	// set app id and topics
	topics, rets, err := topicsAndRets(ci, tps, qoses)
	if err != nil {
		return err
	}

	// check mutex login
	mutexLogin(ci)

	// subscribe the cid topic in nats
	cstr := strconv.FormatInt(ci.id, 10)
	h, err := nc.Subscribe(cstr, func(m *nats.Msg) {
		pub2c(ci, m)
	})
	if err != nil {
		return fmt.Errorf("sub to nats error: %v", err)
	}
	ci.natsHandler = h
	ci.isNatsSubed = true

	// 这里的AppID先设置为CLientId，具体参见connInfo结构
	rpcH, err := getRpc(ci)
	if err != nil {
		return err
	}
	err = rpcH.login(&rpc.LoginMsg{
		Acc:   ci.acc,
		AppID: ci.appID,
		PT:    ci.payloadProtoType,
		Cid:   ci.id,
		Gip:   talents.String2Bytes(ci.rip),
		Ts:    topics,
	})
	if err != nil {
		return fmt.Errorf("login rpc error: %v", err)
	}

	// give back the suback
	pb := proto.NewSubackPacket()
	pb.SetPacketID(pid)

	// return the final qos level
	pb.AddReturnCodes(rets)
	write(ci, pb)

	ci.isSubed = true
	return nil
}
