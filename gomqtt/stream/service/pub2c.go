package service

import (
	"log"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/nats-io/nats"
	"im.zgl/gomqtt/global"
	"im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

type natsInfo struct {
	nc *nats.Conn // connect nats cluster
}

// newnatsInfo return *natsInfo
func newnatsInfo(addrs []string) (*natsInfo, error) {
	// init natsInfo
	nc, err := initnatsInfo(addrs)
	if err != nil {
		return nil, err
	}
	natsInfo := natsInfo{nc: nc}
	return &natsInfo, nil
}

func initnatsInfo(addrs []string) (*nats.Conn, error) {
	nc := initNatsConn(addrs)
	return nc, nil
}

// pushText 推送消息至nats服务
func (ns *natsInfo) pushText(subject string, msg *global.TextMsgs) error {

	b, err := msg.MarshalMsg(nil)
	if err != nil {
		return err
	}

	// log.Println("pushText --- nats", subject, len(msg.Msgs))
	Logger.Info("pushText", zap.String("subject", subject), zap.Int("len", len(msg.Msgs)))

	err = ns.nc.Publish(subject, b)
	return err
}

// pushJson 推送消息至nats服务
func (ns *natsInfo) pushJson(subject string, msg *global.JsonMsgs) error {

	b, err := msg.MarshalMsg(nil)
	if err != nil {
		return err
	}

	Logger.Info("pushJson", zap.String("subject", subject), zap.Int("len", len(msg.Data.Msgs)))

	err = ns.nc.Publish(subject, b)
	return err
}

func initNatsConn(addrs []string) *nats.Conn {
	opts := nats.DefaultOptions
	opts.Servers = addrs
	opts.MaxReconnect = 100
	opts.ReconnectWait = 20 * time.Second
	// opts.NoRandomize = true

	nc, err := opts.Connect()
	if err != nil {
		Logger.Panic("Nats", zap.String("Connect", err.Error()))
		return nil
	}

	// Setup callbacks to be notified on disconnects and reconnects
	nc.Opts.DisconnectedCB = func(nc *nats.Conn) {
		Logger.Info("Nats", zap.String("ConnectedUrl", nc.ConnectedUrl()))
	}

	// See who we are connected to on reconnect.
	nc.Opts.ReconnectedCB = func(nc *nats.Conn) {
		Logger.Info("Nats", zap.String("ConnectedUrl", nc.ConnectedUrl()))
	}

	return nc
}

// taskMsg msg
type taskMsg struct {
	cid         int64
	acc         []byte
	appid       []byte
	payloadType int32
	queue       *Controller
	retChan     chan *CacheRet
	ts          []*proto.Topic
}

var gRetChan chan *CacheRet

func addTask(t *taskMsg) {
	// log.Println("addTask", t)
	gStream.taskChan <- t
}

func startDealTask(taskn int) {

	stopChan := make(chan bool, 1)
	for index := 0; index < taskn; index++ {
		go func() {
			dealTask(gStream.taskChan, stopChan)
		}()
	}
}

func dealTask(ch chan *taskMsg, sc chan bool) {
	//异常接收代码
	defer func() {
		close(ch)
		close(sc)

		log.Println(zap.Stack())
		if err := recover(); err != nil {
			log.Printf("panic: %s\nStack trace:\n%s", err, debug.Stack())
			Logger.Error("startDealTask", zap.Stack())
		}
	}()
	for {
		select {
		case t, ok := <-ch:
			if !ok {
				Logger.Panic("Chan", zap.String("dealTask", "recv chan failed"))
				break
			}
			if t.payloadType == int32(global.PayloadText) {
				PushTextOffLineMsg(t)
			} else if t.payloadType == int32(global.PayloadJson) {
				PushJsonOffLineMsg(t)
			}

			break
		case <-sc:
			Logger.Info("Chan", zap.String("dealTask", "get stop signal"))
			return

		}
	}
}

func PushTextOffLineMsg(t *taskMsg) {
	for _, topicMsg := range t.ts {
		cacheTask := CacheTask{
			MsgTy:   CACHE_TEXT_SELECT,
			TAcc:    t.acc,
			TTopic:  topicMsg.Tp,
			RetChan: t.retChan,
		}
		t.queue.Publish(cacheTask)

		retCache, ok := <-t.retChan
		if !ok {
			Logger.Error("PushOffLineMsg", zap.String("Acc", talents.Bytes2String(t.acc)))
			return
		}
		if retCache.MsgIDs != nil {
			for mmm, msgidMsg := range retCache.MsgIDs.MsgID {
				log.Println(mmm, "----------msgidMsg : ", msgidMsg)
			}
			// push
			MsgsCacha := make([]*global.TextMsg, 0, len(retCache.MsgIDs.MsgID))
			// get Msg
			for _, msgidMsg := range retCache.MsgIDs.MsgID {
				Logger.Info("PushOffLineMsg", zap.String("msgid", talents.Bytes2String(msgidMsg.MsgID)))
				getTask := CacheTask{
					MsgTy:   CACHE_TEXT_GET,
					MsgIDs:  [][]byte{msgidMsg.MsgID},
					RetChan: t.retChan,
				}
				t.queue.Publish(getTask)
				retCache, ok := <-t.retChan
				if !ok {
					Logger.Error("PushOffLineMsg", zap.String("Acc", talents.Bytes2String(t.acc)))
					return
				}
				if retCache.Data != nil {
					var Qos int32
					if msgidMsg.MsgQos <= topicMsg.Qos {
						Qos = msgidMsg.MsgQos
					} else {
						Qos = topicMsg.Qos
					}
					Msg := &global.TextMsg{
						FAcc:       msgidMsg.FAcc,   //t.acc,
						FTopic:     msgidMsg.FTopic, //topicMsg.Tp,
						RetryCount: 3,
						Qos:        Qos,
						MsgID:      msgidMsg.MsgID,
						Msg:        retCache.Data,
					}
					Logger.Info("Push", zap.String("data", talents.Bytes2String(retCache.Data)))
					MsgsCacha = append(MsgsCacha, Msg)
				}
			}
			if len(MsgsCacha) > 0 {
				pushMsg := &global.TextMsgs{
					Msgs: MsgsCacha,
				}
				gStream.nats.pushText(strconv.FormatInt(t.cid, 10), pushMsg)
			}
		}
	}
}

func PushJsonOffLineMsg(t *taskMsg) {
	for _, topicMsg := range t.ts {
		cacheTask := CacheTask{
			MsgTy:   CACHE_JSON_SELECT,
			TAcc:    t.acc,
			TTopic:  topicMsg.Tp,
			RetChan: t.retChan,
		}
		t.queue.Publish(cacheTask)

		retCache, ok := <-t.retChan
		if !ok {
			Logger.Error("PushJsonOffLineMsg", zap.String("Acc", talents.Bytes2String(t.acc)))
			return
		}

		Logger.Info("PushJsonOffLineMsg", zap.Object("retCache", retCache))
		if retCache.MsgIDs != nil {
			for mmm, msgidMsg := range retCache.MsgIDs.MsgID {
				log.Println(mmm, "----------msgidMsg : ", msgidMsg)
			}
			// push
			// MsgsCacha := make([]*global.JsonMsgs, 0, len(retCache.MsgIDs.MsgID))
			pushData := &global.JsonMsgs{
				RetryCount: 3,
				TTopics:    [][]byte{},
				MsgID:      [][]byte{},
			}

			datas := &global.JsonData{
				Msgs: []*global.JsonMsg{},
			}
			// get Msg
			for _, msgidMsg := range retCache.MsgIDs.MsgID {
				Logger.Info("PushJsonOffLineMsg", zap.String("msgid", talents.Bytes2String(msgidMsg.MsgID)))
				getTask := CacheTask{
					MsgTy:   CACHE_JSON_GET,
					MsgIDs:  [][]byte{msgidMsg.MsgID},
					RetChan: t.retChan,
				}
				t.queue.Publish(getTask)
				retCache, ok := <-t.retChan
				if !ok {
					Logger.Error("PushJsonOffLineMsg", zap.String("Acc", talents.Bytes2String(t.acc)))
					return
				}

				if retCache.Data != nil {
					if pushData.Qos <= topicMsg.Qos {
						pushData.Qos = topicMsg.Qos
					}

					sendMsg := &global.JsonMsg{
						FAcc:   talents.Bytes2String(msgidMsg.FAcc),
						FTopic: talents.Bytes2String(msgidMsg.FTopic),
						Type:   int(msgidMsg.MsgTy),
						Time:   int(time.Now().Unix()),
						// @Optimize nick这里先传用户账号,因为发送者不一定和接收者在一台机器上
						Nick:  talents.Bytes2String(msgidMsg.FAcc),
						MsgID: talents.Bytes2String(msgidMsg.MsgID),
						Msg:   talents.Bytes2String(retCache.Data),
					}
					datas.Msgs = append(datas.Msgs, sendMsg)
					pushData.TTopics = append(pushData.TTopics, topicMsg.Tp)
					// log.Println(" to topis is ", string(topicMsg.Tp))
					pushData.MsgID = append(pushData.MsgID, msgidMsg.MsgID)
				}
			}
			pushData.Data = datas
			if len(datas.Msgs) > 0 {
				// push to nats
				err := gStream.nats.pushJson(strconv.FormatInt(t.cid, 10), pushData)
				if err != nil {
					Logger.Error("pushJson", zap.Error(err))
					return
				}
			}
		}
	}
}
