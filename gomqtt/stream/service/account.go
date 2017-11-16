package service

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"im.zgl/gomqtt/global"
	proto "im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

const (
	ONLINE  bool = true  //在线
	OFFLINE bool = false //离线
)

type Accounts struct {
	sync.RWMutex
	Accounts map[string]*Account
}

func NewAccounts() *Accounts {
	as := &Accounts{
		Accounts: make(map[string]*Account),
	}
	return as
}

func (ats *Accounts) GetAccAndAppID(accid, appid []byte) (*Account, *AppID, bool) {
	ats.RLock()
	if acc, ok := ats.Accounts[string(accid)]; ok {
		if appid, ok := acc.AppIDs[string(appid)]; ok {
			ats.RUnlock()
			return acc, appid, true
		}
	}
	ats.RUnlock()
	return nil, nil, false
}

// Login 登陆
func (ats *Accounts) Login(msg *proto.LoginMsg) error {
	ats.Lock()
	var err error
	acc, ok := ats.Accounts[string(msg.Acc)]
	if ok {
		err = acc.Login(msg)
	} else {
		// @ToDo   需要从db中拉取用户的信息，比如好友群信息等
		acc = NewAccount(msg.Acc)
		acc.Login(msg)
		ats.Accounts[string(msg.Acc)] = acc
	}
	ats.Unlock()
	return err
}

// Logout logout
func (ats *Accounts) Logout(accid, appid []byte) error {
	ats.Lock()
	var err error
	acc, ok := ats.Accounts[string(accid)]
	if ok {
		acc.Logout(appid)
	} else {
		err = fmt.Errorf("unfind acc %s, appid %s", talents.Bytes2String(accid), talents.Bytes2String(appid))
	}
	ats.Unlock()
	return err
}

// GetQueueAndRetChan get queue and retchan
func (ats *Accounts) GetQueueAndRetChan(accid, appid []byte) (*Controller, chan *CacheRet, error) {
	ats.RLock()
	acc, ok := ats.Accounts[string(accid)]
	ats.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("unfind acc %s, appid %s", talents.Bytes2String(accid), talents.Bytes2String(appid))
	}
	acc.RLock()
	app, ok := acc.AppIDs[string(appid)]
	acc.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("unfind acc %s, appid %s", talents.Bytes2String(accid), talents.Bytes2String(appid))
	}
	return app.Queue, app.RetChan, nil
}

func (ats *Accounts) PubText(msg *proto.PubTextMsg) error {

	gStream.cache.As.RLock()
	acc, ok := gStream.cache.As.Accounts[string(msg.ToAcc)]
	gStream.cache.As.RUnlock()
	if !ok {
		return fmt.Errorf("unfind acc %s, topic %s", talents.Bytes2String(msg.ToAcc), talents.Bytes2String(msg.Ttp))
	}

	acc.RLock()
	topicMsg, ok := acc.PTopics[string(msg.Ttp)]
	acc.RUnlock()

	if !ok {
		return fmt.Errorf("unfind topic, acc is %s, topic %s", talents.Bytes2String(msg.ToAcc), talents.Bytes2String(msg.Ttp))
	}

	if topicMsg.nastTopic != "0" {
		var Qos int32
		if msg.Qos <= topicMsg.qos {
			Qos = msg.Qos
		} else {
			Qos = topicMsg.qos
		}
		Msg := &global.TextMsg{
			FAcc:       msg.FAcc,
			FTopic:     msg.Ftp,
			RetryCount: 3,
			Qos:        Qos,
			MsgID:      msg.Mid,
			Msg:        msg.Msg,
		}
		msg := &global.TextMsgs{
			Msgs: []*global.TextMsg{Msg},
		}
		// push to nats
		gStream.nats.pushText(topicMsg.nastTopic, msg)
	}

	return nil
}

func (ats *Accounts) PubJson(msg *proto.PubJsonMsg) error {
	gStream.cache.As.RLock()
	acc, ok := gStream.cache.As.Accounts[string(msg.ToAcc)]
	gStream.cache.As.RUnlock()
	if !ok {
		return fmt.Errorf("unfind acc %s, topic %s", talents.Bytes2String(msg.ToAcc), talents.Bytes2String(msg.Ttp))
	}

	acc.RLock()
	topicMsg, ok := acc.PTopics[string(msg.Ttp)]
	acc.RUnlock()

	if !ok {
		return fmt.Errorf("unfind topic, acc is %s, topic %s", talents.Bytes2String(msg.ToAcc), talents.Bytes2String(msg.Ttp))
	}

	if topicMsg.nastTopic != "0" {
		var Qos int32
		if msg.Qos <= topicMsg.qos {
			Qos = msg.Qos
		} else {
			Qos = topicMsg.qos
		}

		sendMsg := &global.JsonMsg{
			FAcc:   talents.Bytes2String(msg.FAcc),
			FTopic: talents.Bytes2String(msg.Ttp),
			Type:   int(msg.MsgType),
			Time:   int(time.Now().Unix()),
			// @Optimize nick这里先传用户账号,因为发送者不一定和接收者在一台机器上
			Nick:  talents.Bytes2String(msg.FAcc),
			MsgID: talents.Bytes2String(msg.Mid),
			Msg:   talents.Bytes2String(msg.Msg),
		}

		datas := &global.JsonData{
			Msgs: []*global.JsonMsg{sendMsg},
		}

		pushData := &global.JsonMsgs{
			RetryCount: 3,
			Qos:        Qos,
			TTopics:    [][]byte{msg.Ttp},
			MsgID:      [][]byte{msg.Mid},
			Data:       datas,
		}
		// push to nats
		err := gStream.nats.pushJson(topicMsg.nastTopic, pushData)
		if err != nil {
			Logger.Error("pushJson", zap.Error(err))
			return err
		}
	}
	return nil

}

type Account struct {
	sync.RWMutex
	Acc    []byte
	AppIDs map[string]*AppID //子用户
	// @ToDo
	// 这里还要保存topic的订阅等级
	STopics map[string]*topicTypeCid
	PTopics map[string]*topicTypeCid
}

// topicTypeCid topic 的类型和订阅该topic的cid （string）类型
type topicTypeCid struct {
	topicTy   int32
	qos       int32
	nastTopic string
}

func newtopicTypeCid(ty int32, qos int32, nastTopic string) *topicTypeCid {
	return &topicTypeCid{topicTy: ty, qos: qos, nastTopic: nastTopic}
}

func NewAccount(Acc []byte) *Account {
	account := &Account{
		Acc:     Acc,
		AppIDs:  make(map[string]*AppID),
		STopics: make(map[string]*topicTypeCid),
		PTopics: make(map[string]*topicTypeCid),
	}
	return account
}

func (acc *Account) NewUser(msg *proto.LoginMsg) error {
	appID := NewAppID()
	appID.Gip = msg.Gip
	appID.Cid = msg.Cid
	appID.Oline = ONLINE
	appID.LastLogin = time.Now().Unix()
	acc.AppIDs[string(msg.AppID)] = appID
	return nil
}

// Login save appid msg and topic msg
func (acc *Account) Login(msg *proto.LoginMsg) error {
	acc.Lock()
	var appID *AppID
	appID, ok := acc.AppIDs[string(msg.AppID)]
	if !ok {
		appID = NewAppID()
		acc.AppIDs[string(msg.AppID)] = appID
	}

	// 通过acc计算出队列
	queue, err := GetQueue(msg.Acc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.Acc)))
		return err
	}
	retChan := make(chan *CacheRet, 10)

	appID.Queue = queue
	appID.RetChan = retChan
	appID.Gip = msg.Gip
	appID.Cid = msg.Cid
	appID.PT = msg.PT
	appID.Oline = ONLINE
	appID.LastLogin = time.Now().Unix()

	// 保存topic 和 cid
	for _, topic := range msg.Ts {
		if global.PChatTopic == topic.Ty {
			// private chat
			topic2Nats := newtopicTypeCid(topic.Ty, topic.Qos, strconv.FormatInt(msg.Cid, 10))
			acc.PTopics[string(topic.Tp)] = topic2Nats
		} else if global.SPushTopic == topic.Ty {
			// single chat
			topic2Nats := newtopicTypeCid(topic.Ty, topic.Qos, strconv.FormatInt(msg.Cid, 10))
			acc.PTopics[string(topic.Tp)] = topic2Nats
		} else if global.BPushTopic == topic.Ty {
			// broad chat
		}
	}

	// save topic
	for _, topic := range msg.Ts {
		appID.Topics[string(topic.Tp)] = topic
	}
	acc.Unlock()
	return nil
}

func (acc *Account) Logout(appid []byte) error {
	acc.Lock()
	appID, ok := acc.AppIDs[string(appid)]
	if !ok {
		acc.Unlock()
		return fmt.Errorf("unfind appID %s", talents.Bytes2String(appid))
	}
	appID.Oline = OFFLINE
	appID.LastLogout = time.Now().Unix()
	// close chan
	close(appID.RetChan)
	// set topic cid offline
	for _, topic := range appID.Topics {
		if topic.Ty == global.SPushTopic {
			if topic2Nats, ok := acc.STopics[string(topic.Tp)]; ok {
				topic2Nats.nastTopic = "0"
			}
		}

		if topic.Ty == global.PChatTopic {
			if topic2Nats, ok := acc.PTopics[string(topic.Tp)]; ok {
				topic2Nats.nastTopic = "0"
			}
		}
	}

	acc.Unlock()

	return nil
}

// Subscribe
func (acc *Account) Subscribe(un []byte, msg *proto.SubMsg) error {
	acc.Lock()
	appID, ok := acc.AppIDs[string(un)]
	if !ok {
		acc.Unlock()
		return fmt.Errorf("unfind appID %s  ", talents.Bytes2String(un))
	} else {
		for _, topic := range msg.Ts {
			appID.Topics[string(topic.Tp)] = topic
		}
	}
	acc.Unlock()

	Logger.Info("Subscribe", zap.String("Gip", fmt.Sprintf("%s", appID.Gip)))
	for _, topic := range appID.Topics {
		Logger.Info("Subscribe", zap.String("Topic", fmt.Sprintf("%s", topic.Tp)))
	}

	return nil
}

// UnSubscribe
func (acc *Account) UnSubscribe(un []byte, msg *proto.UnSubMsg) error {
	acc.Lock()
	appID, ok := acc.AppIDs[string(un)]
	if !ok {
		acc.Unlock()
		return fmt.Errorf("unfind appID %s  ", string(un))
	} else {
		for _, topic := range msg.Ts {
			if _, ok := appID.Topics[string(topic.Tp)]; ok {
				delete(appID.Topics, talents.Bytes2String(topic.Tp))
			}
		}
	}
	acc.Unlock()

	Logger.Info("UnSubscribe", zap.String("Gip", fmt.Sprintf("%s", appID.Gip)))
	for _, topic := range appID.Topics {
		Logger.Info("UnSubscribe", zap.String("Topic", fmt.Sprintf("%s", topic.Tp)))
	}
	return nil
}

func (acc *Account) PubText(facc *Account, appid *AppID, msg *proto.PubTextMsg) error {
	if tycid, ok := acc.STopics[string(msg.Ttp)]; ok {
		// 保存消息
		if tycid.nastTopic != "0" {
			var Qos int32
			if msg.Qos <= tycid.qos {
				Qos = msg.Qos
			} else {
				Qos = tycid.qos
			}
			Msg := &global.TextMsg{
				FAcc:       facc.Acc,
				FTopic:     msg.Ttp,
				RetryCount: 3,
				Qos:        Qos,
				MsgID:      msg.Mid,
				Msg:        msg.Msg,
			}
			msg := &global.TextMsgs{
				Msgs: []*global.TextMsg{Msg},
			}
			// push to nats
			gStream.nats.pushText(tycid.nastTopic, msg)
		}
	}
	return nil
}

func (acc *Account) PubJson(facc *Account, appid *AppID, msg *proto.PubJsonMsg) error {
	if tycid, ok := acc.STopics[string(msg.Ttp)]; ok {
		// 保存消息
		if tycid.nastTopic != "0" {
			var Qos int32
			if msg.Qos <= tycid.qos {
				Qos = msg.Qos
			} else {
				Qos = tycid.qos
			}

			// type JsonMsgs struct {
			// 	RetryCount int32     `msg:"rc"`
			// 	Qos        int32     `msg:"q"`
			// 	TTopics    [][]byte  `msg:"ts"`
			// 	MsgID      [][]byte  `msg:"mis"`
			// 	Data       *JsonData `msg:"d"`
			// }

			// type JsonData struct {
			// 	Msgs []*JsonMsg `json:"msgs"`
			// }

			msg := &global.JsonMsgs{
				RetryCount: 3,
				Qos:        Qos,
				TTopics:    [][]byte{msg.Ttp},
				MsgID:      [][]byte{msg.Mid},
				// Data:       &JsonData{
				// // Msgs: []*JsonMsg{},
				// },
			}
			// push to nats
			gStream.nats.pushJson(tycid.nastTopic, msg)
		}
	}
	return nil
}

// AppID appid
type AppID struct {
	Cid        int64  // 连接版本号
	Gip        []byte // 网关地址
	PT         int32  // payload 协议
	Oline      bool   // 是否在线
	LastLogin  int64  // 最后登录时间
	LastLogout int64  // 最后登出时间
	ApnsToken  []byte // apns token
	Topics     map[string]*proto.Topic
	Queue      *Controller
	RetChan    chan *CacheRet
}

func NewAppID() *AppID {
	appID := &AppID{
		Topics: make(map[string]*proto.Topic),
	}
	return appID
}
