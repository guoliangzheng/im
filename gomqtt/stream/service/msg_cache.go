package service

import (
	"log"
	"sync"

	"im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

// NewMsgCache   key:msgid, value:msg
type MsgCache struct {
	sync.RWMutex
	TextMsgs map[string][]byte
	JsonMsgs map[string][]byte
}

func NewMsgCache() *MsgCache {
	msgcache := &MsgCache{
		TextMsgs: make(map[string][]byte),
		JsonMsgs: make(map[string][]byte),
	}
	return msgcache
}

func (msgCache *MsgCache) TextInsert(msgid []byte, msg []byte) error {
	msgCache.TextMsgs[string(msgid)] = msg
	log.Println("TextInsert msg , msgid is", string(msgid), ",msg is", string(msg))
	return nil
}

func (msgCache *MsgCache) JsonInsert(msgid []byte, msg []byte) error {
	msgCache.JsonMsgs[string(msgid)] = msg
	log.Println("JsonInsert msg , msgid is", string(msgid), ",msg is", string(msg))
	return nil
}

func (msgCache *MsgCache) TextDelete(msgid [][]byte) error {
	for _, id := range msgid {
		delete(msgCache.TextMsgs, talents.Bytes2String(id))
		Logger.Info("TextDelete", zap.String("msgid", talents.Bytes2String(id)))
	}
	return nil
}

func (msgCache *MsgCache) JsonDelete(msgid [][]byte) error {
	for _, id := range msgid {
		delete(msgCache.JsonMsgs, talents.Bytes2String(id))
		Logger.Info("JsonDelete", zap.String("msgid", talents.Bytes2String(id)))
	}
	return nil
}

func (msgCache *MsgCache) TextGet(msgid []byte) ([]byte, bool) {
	if msg, ok := msgCache.TextMsgs[string(msgid)]; ok {
		return msg, true
	}
	return nil, false
}

func (msgCache *MsgCache) JsonGet(msgid []byte) ([]byte, bool) {
	if msg, ok := msgCache.JsonMsgs[string(msgid)]; ok {
		return msg, true
	}
	return nil, false
}

// MsgIdManger 推送消息Id缓存，离线用户用来查看自己是否有消息需要拉取,网关或者消息中心推送的消息id通过acc、topic为键值来存放数据ID
type MsgIdManger struct {
	AccMap map[string]*AccTopicMap
}

func (mim *MsgIdManger) InsertMsgID(facc, ftopic []byte, msg *proto.PubTextMsg) error {
	acc, ok := mim.AccMap[string(msg.ToAcc)]
	if ok {
		tm, ok := acc.TopicMsgID[string(msg.Ttp)]
		if ok {
			msgid := NewMsgID(facc, ftopic, msg)
			tm.MsgID[string(msg.Mid)] = msgid
			log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
		} else {
			tm := NewTopicIDMap()
			acc.TopicMsgID[string(msg.Ttp)] = tm
			msgid := NewMsgID(facc, ftopic, msg)
			tm.MsgID[string(msg.Mid)] = msgid
			log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
		}
	} else {
		acc := NewAccTopicMap()
		mim.AccMap[string(msg.ToAcc)] = acc
		tm := NewTopicIDMap()
		acc.TopicMsgID[string(msg.Ttp)] = tm
		msgid := NewMsgID(facc, ftopic, msg)
		tm.MsgID[string(msg.Mid)] = msgid
		log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
	}
	return nil
}

func (mim *MsgIdManger) InsertJsonMsgID(facc, ftopic []byte, msg *proto.PubJsonMsg) error {
	acc, ok := mim.AccMap[string(msg.ToAcc)]
	if ok {
		tm, ok := acc.TopicMsgID[string(msg.Ttp)]
		if ok {
			msgid := NewJsonMsgID(facc, ftopic, msg)
			tm.MsgID[string(msg.Mid)] = msgid
			log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
		} else {
			tm := NewTopicIDMap()
			acc.TopicMsgID[string(msg.Ttp)] = tm
			msgid := NewJsonMsgID(facc, ftopic, msg)
			tm.MsgID[string(msg.Mid)] = msgid
			log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
		}
	} else {
		acc := NewAccTopicMap()
		mim.AccMap[string(msg.ToAcc)] = acc
		tm := NewTopicIDMap()
		acc.TopicMsgID[string(msg.Ttp)] = tm
		msgid := NewJsonMsgID(facc, ftopic, msg)
		tm.MsgID[string(msg.Mid)] = msgid
		log.Println("InsertMsgID msg , msgid is", string(msgid.MsgID))
	}
	return nil
}

func (mim *MsgIdManger) Len(acc []byte, topic []byte) int {
	accMap, ok := mim.AccMap[string(acc)]
	if ok {
		msgids, ok := accMap.TopicMsgID[string(topic)]
		if ok {
			return len(msgids.MsgID)
		}
	}
	return 0
}

func (mim *MsgIdManger) GetMsgIDs(acc []byte, topic []byte) *TopicIDMap {
	accMap, ok := mim.AccMap[string(acc)]
	if ok {
		msgids, ok := accMap.TopicMsgID[string(topic)]
		if ok {
			return msgids
		}
	}
	return nil
}

func (mim *MsgIdManger) MsgAck(acc []byte, topic []byte, msgid []byte) error {
	accMap, ok := mim.AccMap[string(acc)]
	Logger.Info("MsgAck", zap.String("topic", talents.Bytes2String(topic)), zap.String("msgid", talents.Bytes2String(msgid)))
	if ok {
		if topicMsg, ok := accMap.TopicMsgID[string(topic)]; ok {
			delete(topicMsg.MsgID, talents.Bytes2String(msgid))
			Logger.Info("MsgAck", zap.String("msgid", talents.Bytes2String(msgid)))
		}
	}
	return nil
}

func NewMsgIdManger() *MsgIdManger {
	msgidm := &MsgIdManger{
		AccMap: make(map[string]*AccTopicMap),
	}
	return msgidm
}

// AccTopicMap AccTopicMap里面存放的是各个topic和该topic相关的消息id
type AccTopicMap struct {
	TopicMsgID map[string]*TopicIDMap
}

func NewAccTopicMap() *AccTopicMap {
	tm := &AccTopicMap{
		TopicMsgID: make(map[string]*TopicIDMap),
	}
	return tm
}

// TopicIDMap topic对应的消息idmap
type TopicIDMap struct {
	MsgID map[string]*MsgID
}

func NewTopicIDMap() *TopicIDMap {
	tm := &TopicIDMap{
		MsgID: make(map[string]*MsgID),
	}
	return tm
}

// MsgID 每条消息的具体信息
type MsgID struct {
	MsgTy      int32  // 消息类型
	MsgQos     int32  // 消息Qos
	FAcc       []byte //消息来源Acc
	FTopic     []byte // 消息来源主题
	RetryCount int32  // 消息重发次数
	Expiration int64  // 消息过期时间
	RecvTime   int64  // 消息接收时间
	MsgID      []byte // 消息ID
}

func NewMsgID(facc, ftopic []byte, msg *proto.PubTextMsg) *MsgID {
	msgID := &MsgID{
		// MsgTy:
		// Expiration: msg.Qos,
		// RecvTime:   msg.Qos,
		// RetryCount: msg.RetryCount,
		FAcc:   facc,
		FTopic: ftopic,
		MsgQos: msg.Qos,
		MsgID:  msg.Mid,
	}
	return msgID
}

func NewJsonMsgID(facc, ftopic []byte, msg *proto.PubJsonMsg) *MsgID {
	msgID := &MsgID{
		// MsgTy:
		// Expiration: msg.Qos,
		// RecvTime:   msg.Qos,
		// RetryCount: msg.RetryCount,
		FAcc:   facc,
		FTopic: ftopic,
		MsgQos: msg.Qos,
		MsgID:  msg.Mid,
	}
	return msgID
}

// // NewMsgCache   key:msgid, value:msg
// type MsgCache struct {
// 	sync.RWMutex
// 	Msgs map[string][]byte
// }

// func NewMsgCache() *MsgCache {
// 	msgcache := &MsgCache{
// 		Msgs: make(map[string][]byte),
// 	}
// 	return msgcache
// }

// func (msgCache *MsgCache) Insert(msgid []byte, msg []byte) error {
// 	msgCache.Lock()
// 	msgCache.Msgs[string(msgid)] = msg
// 	log.Println("insert msg , msgid is", string(msgid), ",msg is", string(msg))
// 	msgCache.Unlock()
// 	return nil
// }

// func (msgCache *MsgCache) Delete(msgid []byte) error {
// 	msgCache.Lock()
// 	delete(msgCache.Msgs, talents.Bytes2String(msgid))
// 	msgCache.Unlock()

// 	Logger.Info("Delete", zap.String("msgid", talents.Bytes2String(msgid)))
// 	return nil
// }

// func (msgCache *MsgCache) Get(msgid []byte) ([]byte, bool) {
// 	msgCache.RLock()
// 	if msg, ok := msgCache.Msgs[string(msgid)]; ok {
// 		msgCache.RUnlock()
// 		return msg, true
// 	}
// 	msgCache.RUnlock()
// 	return nil, false
// }

// // MsgIdManger 推送消息Id缓存，离线用户用来查看自己是否有消息需要拉取,网关或者消息中心推送的消息id通过acc、topic为键值来存放数据ID
// type MsgIdManger struct {
// 	sync.RWMutex
// 	AccMap map[string]*AccTopicMap
// }

// func (mim *MsgIdManger) InsertTextMsgID(msg *proto.PubTextMsg) error {
// 	mim.RLock()
// 	acc, ok := mim.AccMap[string(msg.ToAcc)]
// 	mim.RUnlock()

// 	// Logger.Info("InsertTextMsgID", zap.String("ToAcc", talents.Bytes2String(msg.ToAcc)), zap.String("Ttp", talents.Bytes2String(msg.Ttp)), zap.String("msgid", talents.Bytes2String(msg.Mid)))

// 	if ok {
// 		acc.Lock()
// 		tm, ok := acc.TopicMsgID[string(msg.Ttp)]
// 		if ok {
// 			msgid := NewMsgID(msg)
// 			tm.MsgID[string(msg.Mid)] = msgid
// 			log.Println("InsertTextMsgID msg , msgid is", string(msgid.MsgID))
// 		} else {
// 			tm := NewTopicIDMap()
// 			acc.TopicMsgID[string(msg.Ttp)] = tm
// 			msgid := NewMsgID(msg)
// 			tm.MsgID[string(msg.Mid)] = msgid
// 			log.Println("InsertTextMsgID msg , msgid is", string(msgid.MsgID))
// 		}
// 		acc.Unlock()
// 	} else {
// 		acc := NewAccTopicMap()

// 		mim.Lock()
// 		mim.AccMap[string(msg.ToAcc)] = acc
// 		mim.Unlock()

// 		tm := NewTopicIDMap()

// 		acc.Lock()
// 		acc.TopicMsgID[string(msg.Ttp)] = tm
// 		msgid := NewMsgID(msg)
// 		tm.MsgID[string(msg.Mid)] = msgid
// 		log.Println("InsertTextMsgID msg , msgid is", string(msgid.MsgID))
// 		acc.Unlock()
// 	}
// 	return nil
// }

// func (mim *MsgIdManger) Len(acc []byte, topic []byte) int {
// 	mim.RLock()
// 	accMap, ok := mim.AccMap[string(acc)]
// 	mim.RUnlock()
// 	if ok {
// 		accMap.RLock()
// 		msgids, ok := accMap.TopicMsgID[string(topic)]
// 		accMap.RUnlock()
// 		if ok {
// 			return len(msgids.MsgID)
// 		}
// 	}
// 	return 0
// }

// func (mim *MsgIdManger) GetMsgIDs(acc []byte, topic []byte) *TopicIDMap {
// 	mim.RLock()
// 	accMap, ok := mim.AccMap[string(acc)]
// 	mim.RUnlock()
// 	if ok {
// 		accMap.RLock()
// 		msgids, ok := accMap.TopicMsgID[string(topic)]
// 		accMap.RUnlock()
// 		if ok {
// 			return msgids
// 		}
// 	}
// 	return nil
// }

// func (mim *MsgIdManger) TextMsgAck(acc []byte, topic []byte, msgid []byte) error {
// 	mim.RLock()
// 	accMap, ok := mim.AccMap[string(acc)]
// 	mim.RUnlock()
// 	Logger.Info("TextMsgAck", zap.String("topic", talents.Bytes2String(topic)), zap.String("msgid", talents.Bytes2String(msgid)))
// 	if ok {
// 		accMap.Lock()
// 		if topicMsg, ok := accMap.TopicMsgID[string(topic)]; ok {
// 			delete(topicMsg.MsgID, talents.Bytes2String(msgid))
// 			Logger.Info("TextMsgAck", zap.String("msgid", talents.Bytes2String(msgid)))
// 		}
// 		accMap.Unlock()
// 	}
// 	return nil
// }

// func NewMsgIdManger() *MsgIdManger {
// 	msgidm := &MsgIdManger{
// 		AccMap: make(map[string]*AccTopicMap),
// 	}
// 	return msgidm
// }

// // AccTopicMap AccTopicMap里面存放的是各个topic和该topic相关的消息id
// type AccTopicMap struct {
// 	sync.RWMutex
// 	TopicMsgID map[string]*TopicIDMap
// }

// func NewAccTopicMap() *AccTopicMap {
// 	tm := &AccTopicMap{
// 		TopicMsgID: make(map[string]*TopicIDMap),
// 	}
// 	return tm
// }

// // TopicIDMap topic对应的消息idmap
// type TopicIDMap struct {
// 	MsgID map[string]*MsgID
// }

// func NewTopicIDMap() *TopicIDMap {
// 	tm := &TopicIDMap{
// 		MsgID: make(map[string]*MsgID),
// 	}
// 	return tm
// }

// // MsgID 每条消息的具体信息
// type MsgID struct {
// 	MsgTy      int32  // 消息类型
// 	MsgQos     int32  // 消息Qos
// 	RetryCount int32  // 消息重发次数
// 	Expiration int64  // 消息过期时间
// 	RecvTime   int64  // 消息接收时间
// 	MsgID      []byte // 消息ID
// }

// func NewMsgID(msg *proto.PubTextMsg) *MsgID {
// 	msgID := &MsgID{
// 		// MsgTy:
// 		// Expiration: msg.Qos,
// 		// RecvTime:   msg.Qos,
// 		// RetryCount: msg.RetryCount,
// 		MsgQos: msg.Qos,
// 		MsgID:  msg.Mid,
// 	}
// 	return msgID
// }
