package main

import (
	"log"
	"net/url"

	"fmt"

	"github.com/gorilla/websocket"
	proto "im.zgl/gomqtt/mqtt/protocol"
)

func main() {
	log.SetFlags(log.Lshortfile)
	u := url.URL{Scheme: "ws", Host: "localhost:8994", Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

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

	c.WriteMessage(websocket.TextMessage, b)

	_, bb1, err := c.ReadMessage()

	mtype1 := proto.PacketType(bb1[0] >> 4)

	// 根据消息类型创建新的消息结构
	m1, err := mtype1.New()

	//解码消息体
	_, err = m1.Decode(bb1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%T\n", m1)

	// 订阅
	pt1 := proto.NewSubscribePacket()
	pt1.SetPacketID(2)

	pt1.AddTopic([]byte("test1%%1--1000"), byte(1))
	pt1.AddTopic([]byte("test2--2000"), byte(1))

	_, b, err = pt1.Encode()
	if err != nil {
		log.Fatal(err)
	}

	c.WriteMessage(websocket.TextMessage, b)

	_, bb2, err := c.ReadMessage()

	mtype2 := proto.PacketType(bb2[0] >> 4)

	// 根据消息类型创建新的消息结构
	m2, err := mtype2.New()

	//解码消息体
	_, err = m2.Decode(bb2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%T\n", m2)
	// 发布消息
	pt2 := proto.NewPublishPacket()
	pt2.SetPacketID(3)
	pt2.SetQoS(byte(1))
	pt2.SetPayload([]byte("hello paho"))
	pt2.SetTopic([]byte("test2--sunface1"))
	_, b, err = pt2.Encode()
	if err != nil {
		log.Fatal(err)
	}

	c.WriteMessage(websocket.TextMessage, b)
}
