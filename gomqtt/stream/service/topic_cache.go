package service

import (
	"sync"

	"im.zgl/talents"
)

// 对于内存中广播消息ID的存储，需要每台Stream都要保存一份

// 广播需要保存
// stream --->>>>>> topic --- {msgids}
// stream --->>>>>> topic --- {acc, {appids}}

type btCache struct {
	sync.RWMutex
	bts map[string]*bpushAppIDs
}

type bpushAppIDs struct {
	acc   []byte
	appID map[string]int64
}

func newbpushAppIDs() *bpushAppIDs {
	bpushAppIDs := &bpushAppIDs{
		appID: make(map[string]int64),
	}
	return bpushAppIDs
}

func (bc *btCache) get(topic []byte) (*bpushAppIDs, bool) {
	bc.RLock()
	if at, ok := bc.bts[string(topic)]; ok {
		bc.RUnlock()
		return at, ok
	}
	bc.RUnlock()
	return nil, false
}

func (bc *btCache) insert(acc []byte, topic []byte, appid []byte) error {
	bc.Lock()
	if acc, ok := bc.bts[string(topic)]; ok {
		acc.appID[string(appid)] = 0
	} else {
		acc := newbpushAppIDs()
		acc.appID[string(appid)] = 0
		bc.bts[string(topic)] = acc
	}
	bc.Unlock()

	return nil
}

func (bc *btCache) delete(acc []byte, topic []byte, appid []byte) error {
	bc.Lock()
	if acc, ok := bc.bts[string(topic)]; ok {
		delete(acc.appID, talents.Bytes2String(appid))
	}
	bc.Unlock()
	return nil
}

func newbtCache() *btCache {
	return &btCache{bts: make(map[string]*bpushAppIDs)}
}
