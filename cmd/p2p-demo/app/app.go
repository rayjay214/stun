// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements a simple CLI tools to perform NAT traversal via STUN
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/pion/stun/v2/cmd/protocol"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/pion/stun/v2"
)

var server = flag.String("server", "stun.voipgate.com:3478", "Stun server address")
var localKey = flag.String("local", "app1", "local key")
var peerKey = flag.String("peer", "camera1", "peer key")
var signalServer = flag.String("signalServer", "127.0.0.1:8882", "Signal server address")

const (
	udp           = "udp4"
	pingMsg       = "ping"
	pongMsg       = "pong"
	timeoutMillis = 5000
)

var peerAddr *net.UDPAddr

func getCameraAddr(conn *net.TCPConn) {
	var msg protocol.Message
	msg.Header.Prefix = 0x6868
	msg.Header.MsgId = protocol.Msg_0x14
	msg.Header.MsgLen = 20
	msg.Header.Imei = 861193046916228
	msg.Body = new(protocol.Body_0x14)
	body14 := msg.Body.(*protocol.Body_0x14)
	body14.PeerKey = *peerKey
	body14.LocalKey = *localKey
	hex, _ := msg.Encode()
	conn.Write(hex)
	log.Printf("sent to signal server hex %x", protocol.GetHex(hex))
}

func main() { //nolint:gocognit
	flag.Parse()

	c, err := net.Dial("tcp", *signalServer)
	tcpConn := c.(*net.TCPConn)
	if err != nil {
		log.Println("Error connecting to server:", err)
		return
	}
	defer tcpConn.Close()

	go handleTcpConnection(tcpConn)

	srvAddr, err := net.ResolveUDPAddr(udp, *server)

	if err != nil {
		log.Fatalf("Failed to resolve server addr: %s", err)
	}

	conn, err := net.ListenUDP(udp, nil)
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	log.Printf("Listening on %s", conn.LocalAddr())

	var publicAddr stun.XORMappedAddress

	messageChan := listen(conn)

	keepalive := time.Tick(timeoutMillis * time.Millisecond)
	keepaliveMsg := pingMsg

	var quit <-chan time.Time

	gotPong := false
	sentPong := false

	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				log.Println("err")
				return
			}

			switch {
			case string(message) == pingMsg:
				log.Println("receive pingmsg")
				keepaliveMsg = pongMsg

			case string(message) == pongMsg:
				if !gotPong {
					log.Println("Received pong message.")
				}

				// One client may skip sending ping if it receives
				// a ping message before knowning the peer address.
				keepaliveMsg = pongMsg

				gotPong = true

			case stun.IsMessage(message):
				m := new(stun.Message)
				m.Raw = message
				decErr := m.Decode()
				if decErr != nil {
					log.Println("decode:", decErr)
					break
				}
				var xorAddr stun.XORMappedAddress
				if getErr := xorAddr.GetFrom(m); getErr != nil {
					log.Println("getFrom:", getErr)
					break
				}

				if publicAddr.String() != xorAddr.String() {
					log.Printf("My public address: %s\n", xorAddr)
					publicAddr = xorAddr

					//上传本机地址到信令服务器
					var msg protocol.Message
					msg.Header.Prefix = 0x6868
					msg.Header.MsgId = protocol.Msg_0x13
					msg.Header.MsgLen = 34
					msg.Header.Imei = 861193046916228
					msg.Body = new(protocol.Body_0x13)
					body13 := msg.Body.(*protocol.Body_0x13)
					body13.Key = *localKey
					body13.Ip = publicAddr.IP.String()
					body13.Port = uint32(publicAddr.Port)
					hex, _ := msg.Encode()
					_, err = tcpConn.Write(hex)
					if err != nil {
						log.Println("Error sending data to signal server:", err)
						return
					}

					fmt.Println("localkey", *localKey)
					if strings.Contains(*localKey, "app") {
						getCameraAddr(tcpConn)
					}
				}

			default:
				log.Panicln("unknown message", message)
			}

		case <-keepalive:
			// Keep NAT binding alive using STUN server or the peer once it's known
			if peerAddr == nil {
				err = sendBindingRequest(conn, srvAddr)
			} else {
				err = sendStr(keepaliveMsg, conn, peerAddr)
				log.Printf("sent to peerAddr %v\n", peerAddr)
				if keepaliveMsg == pongMsg {
					sentPong = true
				}
			}

			if err != nil {
				log.Panicln("keepalive:", err)
			}

		case <-quit:
			_ = conn.Close()
		}

		if quit == nil && gotPong && sentPong {
			log.Println("Success! Quitting in two seconds.")
			quit = time.After(2 * time.Second)
		}
	}
}

func listen(conn *net.UDPConn) <-chan []byte {
	messages := make(chan []byte)
	go func() {
		for {
			buf := make([]byte, 1024)

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(messages)
				return
			}
			buf = buf[:n]

			messages <- buf
		}
	}()
	return messages
}

func sendBindingRequest(conn *net.UDPConn, addr *net.UDPAddr) error {
	m := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	err := send(m.Raw, conn, addr)
	if err != nil {
		return fmt.Errorf("binding: %w", err)
	}

	return nil
}

func send(msg []byte, conn *net.UDPConn, addr *net.UDPAddr) error {
	_, err := conn.WriteToUDP(msg, addr)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

func sendStr(msg string, conn *net.UDPConn, addr *net.UDPAddr) error {
	return send([]byte(msg), conn, addr)
}

func handleTcpConnection(conn *net.TCPConn) {

	defer conn.Close()

	log.Println("Client: ", conn.RemoteAddr(), " Connected")

	msgbuf := bytes.NewBuffer(make([]byte, 0, 10240))

	databuf := make([]byte, 4096)

	len := 0
	recv := 0

	for {
		//  Read the data
		n, err := conn.Read(databuf)
		if err == io.EOF {
			log.Println("Client ", conn.RemoteAddr(), " exit:")
		}
		if err != nil {
			log.Println("Read error: ", err)
			return
		}
		//  Data is added to the message buffer
		n, err = msgbuf.Write(databuf[:n])
		if err != nil {
			log.Println("Buffer write error: ", err)
			return
		}
		//  Message segmentation loop
		for {
			//  The message header
			tmpdata := msgbuf.Bytes()
			if len == 0 && msgbuf.Len() >= 4 {
				if binary.BigEndian.Uint16(tmpdata[:2]) != 0x6868 {
					log.Println("invalid header")
					return
				}
				len = int(binary.BigEndian.Uint16(tmpdata[3:5]))
				len += 15

				//  Check for long messages
				if len > 10240 {
					log.Println("Message too long")
					return
				}
			}
			//  The message body
			if len > 0 && msgbuf.Len() >= len {
				msg := msgbuf.Next(len)
				recv += 1
				log.Printf(" receive msg %x", protocol.GetHex(msg))
				HandleSignalMsg(msg, conn)
				len = 0
			} else {
				break
			}
		}
	}
}

func HandleSignalMsg(data []byte, conn *net.TCPConn) {
	message := new(protocol.Message)
	err := message.Decode(data)
	if err != nil {
		return
	}
	switch message.Header.MsgId {
	case protocol.Msg_0x94:
		handle0x94(message, conn)
	}
}

func handle0x94(message *protocol.Message, conn *net.TCPConn) {
	body94 := message.Body.(*protocol.Body_0x94)

	peerStr := fmt.Sprintf("%v:%v", body94.Ip, body94.Port)

	var err error
	peerAddr, err = net.ResolveUDPAddr(udp, peerStr)
	if err != nil {
		log.Panicln("resolve peeraddr:", err)
	}
	log.Printf("peerAddr is %v\n", peerAddr)
}
