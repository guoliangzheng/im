package service

import (
	"encoding/binary"
	"fmt"
	"net"

	proto "im.zgl/gomqtt/mqtt/protocol"

	"github.com/gorilla/websocket"
)

// ReadPacket read one packet from conn
func ReadPacket(conn net.Conn) (proto.Packet, []byte, int, error) {
	var (
		// buf for head
		b = make([]byte, 5)

		// total bytes read
		n = 0
	)

	for {
		_, err := conn.Read(b[n : n+1])
		if err != nil {
			return nil, b, 0, err
		}

		// 第一个字节是packet标志位，第二个字节开始为packet body的长度编码，采用的是变长编码
		// 在变长编码中，编码的第二个字节开始为0x80时，表示后面还有字节
		if n >= 1 && b[n] < 0x80 {
			break
		}
		n++

	}

	// fmt.Println("[DEBUG] [ReadPacket] Start -", b)

	// 获取剩余长度
	remLen, _ := binary.Uvarint(b[1 : n+1])
	mtype := proto.PacketType(b[0] >> 4)

	buf := make([]byte, n+1+int(remLen))
	copy(buf, b[:n+1])

	if remLen == 0 {
		msg, err := mtype.New()
		dn, err := msg.Decode(buf)
		if err != nil {
			return nil, buf, 0, err
		}

		return msg, nil, dn, nil
	}

	_, err := conn.Read(buf[n+1:]) //[len(b)+1:]
	if err != nil {
		return nil, buf, 0, err
	}

	msg, err := mtype.New()
	dn, err := msg.Decode(buf)
	if err != nil {
		return nil, buf, 0, err
	}

	return msg, nil, dn, nil
}

// Read a raw message from conn
func Read(conn net.Conn) ([]byte, error) {
	var (
		// the message buffer
		buf []byte

		// tmp buffer to read a single byte
		b = make([]byte, 1)

		// total bytes read
		l = 0
	)

	// Let's read enough bytes to get the message header (msg type, remaining length)
	for {
		// If we have read 5 bytes and still not done, then there's a problem.
		if l > 5 {
			return nil, fmt.Errorf("connect/getMessage: 4th byte of remaining length has continuation bit set")
		}

		n, err := conn.Read(b[0:])
		if err != nil {
			//glog.Debugf("Read error: %v", err)
			return nil, err
		}

		// Technically i don't think we will ever get here
		if n == 0 {
			continue
		}

		buf = append(buf, b...)
		l += n

		// Check the remlen byte (1+) to see if the continuation bit is set. If so,
		// increment cnt and continue reading. Otherwise break.
		if l > 1 && b[0] < 0x80 {
			break
		}
	}

	// Get the remaining length of the message
	remlen, _ := binary.Uvarint(buf[1:])
	buf = append(buf, make([]byte, remlen)...)

	for l < len(buf) {
		n, err := conn.Read(buf[l:])
		if err != nil {
			return nil, err
		}
		l += n
	}

	return buf, nil
}

// WritePacket writes a mqtt packet to a connection
func WritePacket(conn net.Conn, p proto.Packet) error {
	// buf := make([]byte, p.Len())
	_, buf, err := p.Encode()
	if err != nil {
		return err
	}
	_, err = conn.Write(buf)
	return err
}

func ReadWsPacket(conn *websocket.Conn) (proto.Packet, error) {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// 消息类型是包体的第一个字节右移4位
	mtype := proto.PacketType(msg[0] >> 4)

	// 根据消息类型创建新的消息结构
	m, err := mtype.New()

	//解码消息体
	_, err = m.Decode(msg)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func WriteWsPacket(conn *websocket.Conn, p proto.Packet) error {
	_, buf, err := p.Encode()
	if err != nil {
		return err
	}

	err = conn.WriteMessage(websocket.TextMessage, buf)
	return err
}
