package gate

/* 各种业务类型转换集合 */
import (
	"bytes"
	"errors"
	"strconv"

	"im.zgl/gomqtt/global"
	"im.zgl/talents"
)

var topicSep = []byte{'-', '-'}
var appidSep = []byte{'-', '-'}
var plSep = []byte{'%', '%'}

// We need to transfer external user to our internal acc and user*/
func accTrans(ci *connInfo) {
	// check appid
	ts := bytes.Split(ci.cp.Username(), appidSep)
	switch len(ts) {
	case 1: // delay login and subscribe
		ci.acc = ci.cp.Username()
	case 2: // use appid to get topics ,login and subscribe
		ci.appID = ts[1]
		ci.acc = ts[0]
	}
}

func topicTrans(t []byte) ([]byte, int, error) {
	ts := bytes.Split(t, topicSep)
	if len(ts) != 2 {
		return nil, 0, errors.New("topic for subscribe mus contains just one topic type")
	}

	tp := ts[0]

	var ty int
	// check topic type
	s := talents.Bytes2String(ts[1])
	switch s {
	case "1000", "2000", "3000", "4000", "5000":
		ty, _ = strconv.Atoi(s)
	default:
		return nil, 0, errors.New("invalid topic type")
	}

	return tp, ty, nil
}

// set appid and payload proto type
// appid%%pltype--topictype
func appidTrans(ci *connInfo, tp []byte) error {
	ts := bytes.Split(tp, plSep)
	if len(ts) != 2 {
		return errors.New("when use delay login,you must specify payload proto type")
	}

	ci.appID = ts[0]

	plType, _ := strconv.Atoi(string(ts[1]))
	if plType == global.PayloadText || plType == global.PayloadJson || plType == global.PayloadProtobuf {
		ci.payloadProtoType = int32(plType)
		return nil
	}

	return errors.New("invalid payload proto type")
}

func qosTrans(q byte) byte {
	if q > Conf.Mqtt.QosMax {
		q = Conf.Mqtt.QosMax
	}

	return q
}
