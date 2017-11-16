package service

import (
	"fmt"
	"net"

	context "golang.org/x/net/context"

	"google.golang.org/grpc"

	"im.zgl/gomqtt/global"
	"im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

type Rpc struct {
	gs *grpc.Server
}

func NewRpc() *Rpc {
	rpc := &Rpc{}
	return rpc
}

func (rpc *Rpc) Init() {

}

func (rpc *Rpc) Start() {

	// var addr string
	if Conf.GrpcC.Addr == "" {
		Conf.GrpcC.Addr = talents.LocalIP() + ":" + Conf.GrpcC.Port
	}
	Logger.Info("addr", zap.String("addr", Conf.GrpcC.Addr))

	l, err := net.Listen("tcp", Conf.GrpcC.Addr)
	if err != nil {
		Logger.Panic("Init", zap.Error(err))
	}
	rpc.gs = grpc.NewServer()

	proto.RegisterRpcServer(rpc.gs, &Rpc{})
	go rpc.gs.Serve(l)
}

func (r *Rpc) Close() error {
	r.gs.Stop()
	return nil
}

// ---------------- 用户相关接口  ----------------

// Login 登陆
func (rpc *Rpc) Login(ctx context.Context, msg *proto.LoginMsg) (*proto.LoginRet, error) {
	Logger.Info("Login", zap.String("Acc", talents.Bytes2String(msg.Acc)), zap.String("AppID", talents.Bytes2String(msg.AppID)))
	err := gStream.cache.As.Login(msg)
	if err != nil {
		Logger.Error("Login", zap.Error(err), zap.String("Acc", talents.Bytes2String(msg.Acc)), zap.Int64("Cid", msg.Cid))
		return &proto.LoginRet{R: false, M: []byte(fmt.Sprint("%s", err.Error()))}, err
	}

	queue, retChan, err := gStream.cache.As.GetQueueAndRetChan(msg.Acc, msg.AppID)
	if err != nil {
		return &proto.LoginRet{R: false, M: []byte(fmt.Sprintf("%s", err.Error()))}, err
	}

	task := &taskMsg{
		cid:         msg.Cid,
		acc:         msg.Acc,
		appid:       msg.AppID,
		payloadType: msg.PT,
		queue:       queue,
		retChan:     retChan,
		ts:          msg.Ts,
	}
	addTask(task)
	return &proto.LoginRet{R: true}, nil
}

// Logout 登出
func (rpc *Rpc) Logout(ctx context.Context, msg *proto.LogoutMsg) (*proto.LogoutRet, error) {
	err := gStream.cache.As.Logout(msg.Acc, msg.AppID)
	if err != nil {
		return &proto.LogoutRet{R: false, M: []byte(fmt.Sprint("%s", err.Error()))}, nil
	}
	return &proto.LogoutRet{R: true}, nil
}

// ---------------- 订阅相关接口  ----------------

// Subscribe 订阅
func (rpc *Rpc) Subscribe(ctx context.Context, msg *proto.SubMsg) (*proto.SubRet, error) {

	return &proto.SubRet{R: true}, nil
}

// UnSubscribe 取消订阅
func (rpc *Rpc) UnSubscribe(ctx context.Context, msg *proto.UnSubMsg) (*proto.UnSubRet, error) {

	return &proto.UnSubRet{R: true}, nil
}

// PubAck  puback
func (rpc *Rpc) PubAck(ctx context.Context, msg *proto.PubAckMsg) (*proto.PubAckRet, error) {
	// 通过acc计算出队列
	queue, err := GetQueue(msg.Acc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.Acc)))
		return &proto.PubAckRet{R: false, M: []byte(err.Error())}, err
	}

	MsgIDs := make([][]byte, len(msg.Mids), len(msg.Mids))
	var ttopic []byte
	for index, msgidMsg := range msg.Mids {
		MsgIDs[index] = msgidMsg.Mid
		ttopic = msgidMsg.Tp
		Logger.Info("PubAck", zap.String("msgid", talents.Bytes2String(msgidMsg.Mid)))
	}

	cacheTask := CacheTask{
		TAcc:   msg.Acc,
		TTopic: ttopic,
		MsgIDs: MsgIDs,
	}

	if msg.Plty == global.PayloadText {
		cacheTask.MsgTy = CACHE_TEXT_DELETE
	} else if msg.Plty == global.PayloadJson {
		cacheTask.MsgTy = CACHE_JSON_DELETE
	}

	queue.Publish(cacheTask)

	return &proto.PubAckRet{R: true}, nil
}

// PubText
func (rpc *Rpc) PubText(ctx context.Context, msg *proto.PubTextMsg) (*proto.PubTextRet, error) {

	// compute queue by acc
	queue, err := GetQueue(msg.ToAcc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.ToAcc)))
		return &proto.PubTextRet{R: false, M: []byte(err.Error())}, nil
	}

	cacheTask := CacheTask{
		MsgTy:  CACHE_TEXT_INSERT,
		FAcc:   msg.FAcc,
		FTopic: msg.Ftp,
		TAcc:   msg.ToAcc,
		TTopic: msg.Ttp,
		Msg:    msg,
		MsgIDs: [][]byte{msg.Mid},
	}
	queue.Publish(cacheTask)
	// push data
	err = gStream.cache.As.PubText(msg)
	if err != nil {
		Logger.Error("PubText", zap.Error(err))
		return &proto.PubTextRet{R: false, M: []byte(fmt.Sprint("%s", err.Error()))}, nil
	}

	return &proto.PubTextRet{R: true}, nil
}

// PubJson json格式推送
func (rpc *Rpc) PubJson(ctx context.Context, msg *proto.PubJsonMsg) (*proto.PubJsonRet, error) {
	// 通过acc计算出队列
	queue, err := GetQueue(msg.ToAcc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.ToAcc)))
		return &proto.PubJsonRet{R: false, M: []byte(err.Error())}, nil
	}

	Logger.Info("PubJson", zap.String("to topic", string(msg.Ttp)), zap.String("ToAcc", string(msg.ToAcc)))
	cacheTask := CacheTask{
		MsgTy:   CACHE_JSON_INSERT,
		FAcc:    msg.FAcc,
		FTopic:  msg.Ftp,
		TAcc:    msg.ToAcc,
		TTopic:  msg.Ttp,
		JsonMsg: msg,
		MsgIDs:  [][]byte{msg.Mid},
	}

	Logger.Info("JsonMsgInsert", zap.String("Mid", string(msg.Mid)))

	queue.Publish(cacheTask)

	if err := gStream.cache.As.PubJson(msg); err != nil {
		return &proto.PubJsonRet{R: false, M: []byte(err.Error())}, nil
	}
	return &proto.PubJsonRet{R: true}, nil
}

// SetAppID 设置AppID
// func (rpc *Rpc) SetAppID(ctx context.Context, msg *proto.AppIDMsg) (*proto.AppIDRet, error) {
// 	if acc, ok := gStream.cache.Cids.get(msg.Cid); ok {
// 		err := acc.acc.SetAppID(acc.user, msg)
// 		if err != nil {
// 			log.Println("SetAppID err ", err)
// 			return &proto.AppIDRet{R: false, M: []byte(fmt.Sprint("%s", err.Error()))}, err
// 		}
// 	}
// 	return &proto.AppIDRet{R: false, M: []byte("UnSubscribe 成功调用")}, nil
// }

// // BPull 拉取广播推送
// func (rpc *Rpc) BPull(ctx context.Context, msg *proto.BPushMsg) (*proto.BPushRet, error) {

// 	return &proto.BPushRet{Msg: []byte("BPull 成功调用")}, nil
// }

// // SPull 拉取单播推送
// func (rpc *Rpc) SPull(ctx context.Context, msg *proto.SPushMsg) (*proto.SPushRet, error) {

// 	return &proto.SPushRet{Msg: []byte("SPull 成功调用")}, nil
// }

// // PPull 拉取私聊
// func (rpc *Rpc) PPull(ctx context.Context, msg *proto.PChatMsg) (*proto.PChatRet, error) {

// 	return &proto.PChatRet{Msg: []byte("PPull 成功调用")}, nil
// }

// // GPull 拉取群聊
// func (rpc *Rpc) GPull(ctx context.Context, msg *proto.GChatMsg) (*proto.GChatRet, error) {

// 	return &proto.GChatRet{Msg: []byte("GPull 成功调用")}, nil
// }

// // 用户设置相关接口

// // SetNick 设置昵称
// func (rpc *Rpc) SetNick(ctx context.Context, msg *proto.NickMsg) (*proto.NickRet, error) {

// 	return &proto.NickRet{Msg: []byte("SetNick 成功调用")}, nil
// }

// // SetApns 设置Apns
// func (rpc *Rpc) SetApns(ctx context.Context, msg *proto.ApnsMsg) (*proto.ApnsRet, error) {

// 	return &proto.ApnsRet{Msg: []byte("SetApns 成功调用")}, nil
// }

// // SetLabel Label
// func (rpc *Rpc) SetLabel(ctx context.Context, msg *proto.LabelMsg) (*proto.LabelRet, error) {

// 	return &proto.LabelRet{Msg: []byte("SetLabel 成功调用")}, nil
// }

// ---------------- 消息推送相关接口  ----------------

// // BPush 广播推送
// func (rpc *Rpc) BPush(ctx context.Context, bm *proto.BPushMsg) (*proto.BPushRet, error) {

// 	return &proto.BPushRet{}, nil
// }

// // SPush 单播推送
// func (rpc *Rpc) SPush(ctx context.Context, sm *proto.SPushMsg) (*proto.SPushRet, error) {

// 	return &proto.SPushRet{}, nil
// }

// // PChat 私聊
// func (rpc *Rpc) PChat(ctx context.Context, pm *proto.PChatMsg) (*proto.PChatRet, error) {

// 	return &proto.PChatRet{}, nil
// }

// // GChat 群聊
// func (rpc *Rpc) GChat(ctx context.Context, gm *proto.GChatMsg) (*proto.GChatRet, error) {

// 	return &proto.GChatRet{}, nil
// }
