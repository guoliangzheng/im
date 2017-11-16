package main

import (
	"log"
	"net"
	"time"

	proto "im.zgl/gomqtt/mqtt/protocol"
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "localhost:8993")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// 连接
	pt := proto.NewConnectPacket()
	pt.SetClientId([]byte("id21083536524661"))
	pt.SetPacketID(1)
	pt.SetUsername([]byte("sunface"))
	pt.SetPassword([]byte("123456"))
	pt.SetVersion(0x4)
	pt.SetKeepAlive(20)

	_, b, err := pt.Encode()
	if err != nil {
		log.Fatal(err)
	}

	conn.Write(b)

	// 订阅
	pt1 := proto.NewSubscribePacket()
	pt1.SetPacketID(2)

	pt1.AddTopic([]byte("test1%%1--1000"), byte(1))
	pt1.AddTopic([]byte("test2--2000"), byte(1))

	_, b, err = pt1.Encode()
	if err != nil {
		log.Fatal(err)
	}

	conn.Write(b)

	time.Sleep(60 * time.Second)
}
