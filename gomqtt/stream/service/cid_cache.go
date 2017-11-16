package service

import (
	"log"
	"sync"

	"im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

type conIDs struct {
	sync.RWMutex
	cids map[int64]*accMsg
}

func newconIDs() *conIDs {
	conIDs := &conIDs{
		cids: make(map[int64]*accMsg),
	}
	return conIDs
}

// accMsg 存放acccound和appid的映射关系,主题下面保存该用户的信息
type accMsg struct {
	acc       []byte
	appID     []byte
	payloadty int32
	queue     *Controller
	retChan   chan *CacheRet
}

func newaccMsg() *accMsg {
	accmsg := &accMsg{}
	return accmsg
}

func (cids *conIDs) add(msg *proto.LoginMsg) error {
	// // 通过acc计算出队列
	queue, err := GetQueue(msg.Acc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.Acc)))
		return err
	}

	cids.Lock()
	if acc, ok := cids.cids[msg.Cid]; ok {
		acc.appID = msg.AppID
		acc.payloadty = msg.PT
		acc.queue = queue
		if acc.retChan == nil {
			acc.retChan = make(chan *CacheRet, 10)
		}
	} else {
		acc := newaccMsg()
		acc.acc = msg.Acc
		acc.appID = msg.AppID
		acc.payloadty = msg.PT
		acc.queue = queue
		acc.retChan = make(chan *CacheRet, 10)
		cids.cids[msg.Cid] = acc
	}
	cids.Unlock()
	return nil
}

func (cids *conIDs) addAndRetQueueChan(msg *proto.LoginMsg) (*Controller, chan *CacheRet, error) {
	// // 通过acc计算出队列
	queue, err := GetQueue(msg.Acc)
	if err != nil {
		Logger.Error("GetQueue", zap.Error(err), zap.String("acc", talents.Bytes2String(msg.Acc)))
		return nil, nil, err
	}
	retChan := make(chan *CacheRet, 10)
	cids.Lock()
	acc, ok := cids.cids[msg.Cid]
	if ok {
		acc.appID = msg.AppID
		acc.queue = queue
		acc.retChan = retChan
	} else {
		acc = newaccMsg()
		acc.acc = msg.Acc
		acc.appID = msg.AppID
		acc.queue = queue
		acc.retChan = retChan
		cids.cids[msg.Cid] = acc
	}
	cids.Unlock()
	return queue, retChan, nil
}

func (cids *conIDs) get(cid int64) (*accMsg, bool) {
	cids.RLock()
	if acc, ok := cids.cids[cid]; ok {
		cids.RUnlock()
		return acc, true
	}
	cids.RUnlock()
	return nil, false
}

func (cids *conIDs) delete(cid int64) bool {
	cids.Lock()
	if acc, ok := cids.cids[cid]; ok {
		// close chan
		if acc.retChan != nil {
			log.Println("Close  ", &acc.retChan)
			close(acc.retChan)
		}
		delete(cids.cids, cid)
		cids.Unlock()
		return true
	}
	cids.Unlock()
	return false
}

type conID struct {
	acc *Account
}
