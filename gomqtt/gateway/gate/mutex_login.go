package gate

/* 互斥登录模块
4种互斥登录模式，通过mutex.type进行配置*/
import (
	"sync"

	"im.zgl/talents"
)

type account struct {
	ci    *connInfo
	users map[string]*connInfo
}

var accounts = make(map[string]*account)
var al = &sync.RWMutex{}

func mutexLogin(ci *connInfo) {
	switch Conf.Mutex.Type {
	case 1: // mutex login disable

	case 2: // mutex with account(Username)
		al.RLock()
		acc, ok := accounts[string(ci.acc)]
		al.RUnlock()

		if ok {
			// kick off the former connections
			closeConn(acc.ci)

			// wait for kick off done
			<-acc.ci.relogin
		}

		//update now one
		al.Lock()
		accounts[string(ci.acc)] = &account{
			ci: ci,
		}
		al.Unlock()

	case 3: //mutex with account + appid(Username + appid)
		al.RLock()
		acc, ok := accounts[string(ci.acc)]
		al.RUnlock()

		if ok {
			c, ok := acc.users[string(ci.appID)]
			if ok {
				closeConn(c)

				<-c.relogin
			}

			acc.users[string(ci.appID)] = ci
		} else {
			accounts[string(ci.acc)] = &account{
				users: map[string]*connInfo{
					talents.Bytes2String(ci.appID): ci,
				},
			}
		}

	case 4: //mutex with clientid
		al.RLock()
		acc, ok := accounts[string(ci.cp.ClientId())]
		al.RUnlock()

		if ok {
			// kick off the former connections
			closeConn(acc.ci)

			// wait for kick off done
			<-acc.ci.relogin
		}

		//update now one
		al.Lock()
		accounts[string(ci.cp.ClientId())] = &account{
			ci: ci,
		}
		al.Unlock()

		return
	}

}

func delMutex(ci *connInfo) {
	switch Conf.Mutex.Type {
	case 1:

	case 2:
		al.Lock()
		delete(accounts, talents.Bytes2String(ci.acc))
		al.Unlock()

	case 3:
		al.Lock()
		acc, _ := accounts[string(ci.acc)]
		delete(acc.users, talents.Bytes2String(ci.appID))
		al.Unlock()
	case 4:
		al.Lock()
		delete(accounts, talents.Bytes2String(ci.cp.ClientId()))
		al.Unlock()
	}
}
