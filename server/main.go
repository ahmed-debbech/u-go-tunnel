package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CurrentUserConn struct {
	Id   string
	Conn net.Conn
}

type Tunnel struct {
	ConnectorConn       net.Conn
	FromUserToConnector chan Frame
}

var (
	UserConns   = make(map[uint32]chan Frame) //Every connected user socket to server with its random id
	MuUserConns sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("SERVER")

	incomingReq := make(chan net.Conn, 100)
	connectorCh := make(chan net.Conn, 10)

	go ListenForConnectors(connectorCh)
	go RegisterConnectors(connectorCh)
	go ListenIncomingConns(9001, incomingReq)
	go ProcessUsers(incomingReq)

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

func RegisterConnectors(connectorCh chan net.Conn) {

	for conn := range connectorCh {
		fromUserToConnector := make(chan Frame, 1000)

		// Generate unique Connector ID
		id := GenerateNewConnectorId(conn, fromUserToConnector)

		log.Println("New connector linked")

		_, err := ReadConnectorConfig(conn, id)
		if err != nil {
			log.Println("could not get port config for connector", id)
			CleanUpConnector(ConnectorConnection, id)
			continue
		}

		log.Println(ExposedPorts)
		log.Println("read connector config file successfully")

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer func() {
				CleanUpConnector(ConnectorConnection, id)
			}()

			for {
				select {
				case <-ctx.Done():
					return
				default:

					frame, err := ParseFrame(conn)
					if err != nil {
						log.Println("Read to connector failed:", err)
						cancel()
						return
					}

					MuUserConns.Lock()
					if ch, ok := UserConns[frame.ConnId]; ok {
						ch <- frame
					}
					MuUserConns.Unlock()
				}
			}
		}()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case frame := <-fromUserToConnector:
					dat := SerializeFrame(frame)
					if _, err := ConnectorConnection[id].ConnectorConn.Write(dat); err != nil {
						log.Println("Write to connector failed:", err)
						cancel()
						return
					}
				}

			}
		}()
	}
}

func ProcessUsers(incomingReq chan net.Conn) {
	for userConn := range incomingReq {
		outReq := make(chan Frame, 1000)
		ctx, cancel := context.WithCancel(context.Background())

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

		log.Println("New user connection with ID:", id)

		// Read from user
		go func(id uint32, userConn net.Conn) {
			defer func() {
				userConn.Close()
				MuUserConns.Lock()
				delete(UserConns, id)
				MuUserConns.Unlock()
				close(outReq)
				log.Println(id, "connection to user closed by user")
			}()

			idBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(idBuf, id)

			for {
				select {
				case <-ctx.Done():
					return
				default:
					buff := make([]byte, 1024)
					s := strings.Split(userConn.RemoteAddr().String(), ":")[1]
					portN, _ := strconv.Atoi(s)
					n, err := userConn.Read(buff)
					if err != nil {
						if err != io.EOF {
							log.Println(id, "could not read from user:", err)
						}
						frame := ConstructFrame(id, uint16(portN), []byte{}) // the closing frame for this connection
						if ConnectorId, ok := ExposedPorts[uint16(portN)]; ok {
							ConnectorConnection[ConnectorId].FromUserToConnector <- frame
						}
						cancel()
						return
					}
					frame := ConstructFrame(id, uint16(portN), buff[:n])
					if ConnectorId, ok := ExposedPorts[uint16(portN)]; ok {
						ConnectorConnection[ConnectorId].FromUserToConnector <- frame
					}
				}
			}
		}(id, userConn)

		// Write back to user
		go func(id uint32, userConn net.Conn, outReq chan Frame) {
			for {
				select {
				case <-ctx.Done():
					return
				case frame := <-outReq:
					MuUserConns.Lock()
					_, ok := UserConns[frame.ConnId]
					MuUserConns.Unlock()
					if !ok {
						cancel()
						return
					}

					if frame.Length > 0 {
						if _, err := userConn.Write(frame.Data); err != nil {
							log.Println(id, "write to user failed:", err)
							cancel()
							return
						}
					} else {
						log.Println(id, "App requested to close user connection")
						userConn.Close()
						cancel()
						return
					}
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
