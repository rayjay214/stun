package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/pion/stun/v2/cmd/protocol"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8882", "listen address")
	flag.Parse()

	tcpAddr, err := net.ResolveTCPAddr("tcp", listenAddr)
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println("Listen Error: ", err)
		return
	}
	for {
		conn, err := ln.Accept()
		tcpConn := conn.(*net.TCPConn)
		if err != nil {
			log.Println("Accept Error: ", err)
			continue
		}
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(time.Second * 60)

		go handleConnection(tcpConn)
	}
}

func handleConnection(conn *net.TCPConn) {

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
				log.Printf(" receive msg %x", GetHex(msg))
				HandleMsg(msg, conn)
				len = 0
			} else {
				break
			}
		}
	}
}

var mpConn sync.Map
var mpAddr sync.Map

func HandleMsg(data []byte, conn *net.TCPConn) {
	message := new(protocol.Message)
	err := message.Decode(data)
	if err != nil {
		return
	}
	switch message.Header.MsgId {
	case protocol.Msg_0x13:
		handle0x13(message, conn)
	case protocol.Msg_0x14:
		handle0x14(message, conn)
	}
}

func handle0x13(message *protocol.Message, conn *net.TCPConn) {
	body13 := message.Body.(*protocol.Body_0x13)

	mpConn.Store(body13.Key, conn)

	strAddr := fmt.Sprintf("%v:%v", body13.Ip, body13.Port)
	mpAddr.Store(body13.Key, strAddr)
	fmt.Printf("save %v:%v\n", body13.Key, strAddr)
}

func handle0x14(message *protocol.Message, conn *net.TCPConn) {
	body14 := message.Body.(*protocol.Body_0x14)

	fmt.Printf("handle 0x14 %v\n", body14)

	addr, found := mpAddr.Load(body14.PeerKey)
	strAddr := addr.(string)

	msg94 := new(protocol.Message)
	msg94.Header = message.Header
	msg94.Body = new(protocol.Body_0x94)
	body94 := msg94.Body.(*protocol.Body_0x94)
	if found {
		addrArr := strings.Split(strAddr, ":")
		body94.Ip = addrArr[0]
		port, _ := strconv.Atoi(addrArr[1])
		body94.Port = uint32(port)
	} else {
		body94.Ip = ""
		body94.Port = 0
	}
	body94.Key = body14.PeerKey
	data94ToSend, _ := msg94.Encode()
	fmt.Printf("send to app %x\n", protocol.GetHex(data94ToSend))
	conn.Write(data94ToSend)

	//将APP的地址发给Camera
	appAddr, found := mpAddr.Load(body14.LocalKey)
	strAppAddr := appAddr.(string)
	body94_camera := msg94.Body.(*protocol.Body_0x94)
	if found {
		addrArr := strings.Split(strAppAddr, ":")
		body94_camera.Ip = addrArr[0]
		port, _ := strconv.Atoi(addrArr[1])
		body94_camera.Port = uint32(port)
	} else {
		body94_camera.Ip = ""
		body94_camera.Port = 0
	}
	body94_camera.Key = body14.LocalKey
	data94CameraToSend, _ := msg94.Encode()
	tConn, found := mpConn.Load(body14.PeerKey)
	cameraConn := tConn.(*net.TCPConn)
	fmt.Printf("send to camera %x\n", protocol.GetHex(data94CameraToSend))
	cameraConn.Write(data94CameraToSend)
}

func GetHex(data []byte) []byte {
	dstEncode := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(dstEncode, data)
	rst, err := hex.DecodeString(string(dstEncode))
	if err != nil {
		return nil
	} else {
		return rst
	}
}
