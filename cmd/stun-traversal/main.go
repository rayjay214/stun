// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements a simple CLI tools to perform NAT traversal via STUN
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pion/stun/v2"
)

var server = flag.String("server", "stun.voipgate.com:3478", "Stun server address") //nolint:gochecknoglobals

const (
	udp           = "udp4"
	pingMsg       = "ping"
	pongMsg       = "pong"
	timeoutMillis = 5000
)

func main() { //nolint:gocognit
	flag.Parse()

	srvAddr, err := net.ResolveUDPAddr(udp, *server)
	//srvAddr, err := net.ResolveUDPAddr(udp, "stun.voipgate.com:3478")
	//srvAddr, err := net.ResolveUDPAddr(udp, "114.215.190.173:3478")
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
	var peerAddr *net.UDPAddr

	messageChan := listen(conn)
	var peerAddrChan <-chan string

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

					peerAddrChan = getPeerAddr()
				}

			default:
				log.Panicln("unknown message", message)
			}

		case peerStr := <-peerAddrChan:
			peerAddr, err = net.ResolveUDPAddr(udp, peerStr)
			if err != nil {
				log.Panicln("resolve peeraddr:", err)
			}
			log.Printf("peerAddr is %v\n", peerAddr)

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

func getPeerAddr() <-chan string {
	result := make(chan string)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		log.Println("Enter remote peer address:")
		peer, _ := reader.ReadString('\n')
		result <- strings.Trim(peer, " \r\n")
	}()

	return result
}

func listen(conn *net.UDPConn) <-chan []byte {
	messages := make(chan []byte)
	go func() {
		for {
			buf := make([]byte, 1024)

			n, _, err := conn.ReadFromUDP(buf)
			//log.Printf("read %v bytes %v", n, string(buf[:n]))
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
