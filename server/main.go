package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

type CurrentUserConn struct {
	Id   string
	Conn net.Conn
}

var (
	ConnectorConnection net.Conn
	connectorMu         sync.RWMutex

	UserConns   = make(map[uint32]chan []byte)
	MuUserConns sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("SERVER")

	incomingReq := make(chan net.Conn, 100)
	fromUserToConnector := make(chan []byte, 1000)
	connectorCh := make(chan net.Conn, 10)

	go ListenForConnectors(connectorCh)
	go RegisterConnectors(connectorCh, fromUserToConnector)
	go ListenIncomingConns(9001, incomingReq)
	go ProcessUsers(incomingReq, fromUserToConnector)

	select {}
}

func ListenForConnectors(connectorCh chan net.Conn) {
	listener, err := net.Listen("tcp", ":2221")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		connectorCh <- conn
	}
}

func RegisterConnectors(connectorCh chan net.Conn, fromUserToConnector chan []byte) {
	conn := <-connectorCh
	connectorMu.Lock()
	ConnectorConnection = conn
	connectorMu.Unlock()

	log.Println("New connector linked")

	go func() {
		for {
			idBuf := make([]byte, 4)
			if _, err := io.ReadFull(conn, idBuf); err != nil {
				log.Println("Read TAG failed:", err)
				return
			}
			tag := binary.BigEndian.Uint32(idBuf)

			lenBuf := make([]byte, 4)
			if _, err := io.ReadFull(conn, lenBuf); err != nil {
				log.Println("Read LENGTH failed:", err)
				return
			}
			length := binary.BigEndian.Uint32(lenBuf)

			data := make([]byte, length)
			if _, err := io.ReadFull(conn, data); err != nil {
				log.Println("Read DATA failed:", err)
				return
			}

			MuUserConns.Lock()
			if ch, ok := UserConns[tag]; ok {
				ch <- data
			}
			MuUserConns.Unlock()
		}
	}()

	go func() {
		for data := range fromUserToConnector {
			connectorMu.RLock()
			if ConnectorConnection != nil {
				if _, err := ConnectorConnection.Write(data); err != nil {
					log.Println("Write to connector failed:", err)
				}
			}
			connectorMu.RUnlock()
		}
	}()
}

func ProcessUsers(incomingReq chan net.Conn, fromUserToConnector chan []byte) {
	for userConn := range incomingReq {
		outReq := make(chan []byte, 1000)

		// Generate unique user ID
		var id uint32
		for {
			id = rand.Uint32()
			MuUserConns.Lock()
			if _, exists := UserConns[id]; !exists {
				UserConns[id] = outReq
				MuUserConns.Unlock()
				break
			}
			MuUserConns.Unlock()
		}

		log.Println("New user ID:", id)

		// Read from user
		go func(id uint32, userConn net.Conn) {
			defer func() {
				userConn.Close()
				MuUserConns.Lock()
				delete(UserConns, id)
				MuUserConns.Unlock()
				close(outReq)
				log.Println(id, "user connection closed")
			}()

			idBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(idBuf, id)

			for {
				buff := make([]byte, 1024)
				n, err := userConn.Read(buff)
				if err != nil {
					if err != io.EOF {
						log.Println(id, "read error:", err)
					}
					return
				}

				lenBuf := make([]byte, 4)
				binary.BigEndian.PutUint32(lenBuf, uint32(n))

				frame := append(idBuf, lenBuf...)
				frame = append(frame, buff[:n]...)
				fromUserToConnector <- frame
			}
		}(id, userConn)

		// Write back to user
		go func(id uint32, userConn net.Conn, outReq chan []byte) {
			for data := range outReq {
				if _, err := userConn.Write(data); err != nil {
					log.Println(id, "write to user failed:", err)
					return
				}
			}
		}(id, userConn, outReq)
	}
}

func ListenIncomingConns(port int, incomingReq chan net.Conn) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		incomingReq <- conn
	}
}
