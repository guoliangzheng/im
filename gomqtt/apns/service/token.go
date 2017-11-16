package service

import (
	"sync"

	"github.com/nats-io/nats"
)

// Acc -> AppID -> Token
// BTopic ->  Token
// Token -> Acc + AppID
type TokenManager struct {
	*sync.RWMutex
	Acc2Token    map[string]*Appid2Token
	BTopic2Token map[string][]byte
	Token2acc    map[string]*AccAndToken
}

type Appid2Token struct {
	AT map[string][]byte
}

type AccAndToken struct {
	Acc   []byte
	Token []byte
}

func setToken(msg *nats.Msg) {

}

func delToken(msg *nats.Msg) {

}
