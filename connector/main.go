package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var (
	ServerIp    = "127.0.0.1:2221"
	MuUserConns sync.Mutex
	UserConns   = make(map[uint32]net.Conn)
)

func main() {
	log.Println("CONNECTOR")

	outgoingReq := make(chan []byte, 1000)

	conn := ConnectToServer()

	go ReadFromServer(conn, outgoingReq)
	go WriteToServer(conn, outgoingReq)

	select {}
}

// ConnectToServer establishes TCP connection to the server
func ConnectToServer() net.Conn {
	for {
		conn, err := net.Dial("tcp", ServerIp)
		if err != nil {
			log.Println("Cannot connect to server, retrying in 2s:", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Println("Connected to server")
		return conn
	}
}

// ConnectToApp establishes TCP connection to an app on a given port
func ConnectToApp(userTag uint32, port string) net.Conn {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		log.Println("Cannot connect to app:", err)
		return nil
	}
	log.Println(userTag, "New connection to app on port:", port)
	return conn
}

// ReadFromServer reads frames from server and forwards to apps
func ReadFromServer(conn net.Conn, outgo chan []byte) {
	for {
		idBuf := make([]byte, 4)
		if _, err := ReadFull(conn, idBuf); err != nil {
			log.Println("Read TAG failed:", err)
			return
		}
		tag := binary.BigEndian.Uint32(idBuf)

		lenBuf := make([]byte, 4)
		if _, err := ReadFull(conn, lenBuf); err != nil {
			log.Println(tag, "Read LENGTH failed:", err)
			return
		}
		length := binary.BigEndian.Uint32(lenBuf)

		data := make([]byte, length)
		if _, err := ReadFull(conn, data); err != nil {
			log.Println(tag, "Read DATA failed:", err)
			return
		}

		MuUserConns.Lock()
		appConn, ok := UserConns[tag]
		MuUserConns.Unlock()

		if !ok || appConn == nil {
			appConn = ConnectToApp(tag, "80")
			if appConn == nil {
				log.Println(tag, "Cannot connect to app, skipping frame")
				continue
			}

			MuUserConns.Lock()
			UserConns[tag] = appConn
			MuUserConns.Unlock()

			go func(tag uint32, conn net.Conn) {
				for {
					data, ok := ReadFromApp(conn)
					if !ok {
						log.Println(tag, "App connection closed")
						MuUserConns.Lock()
						delete(UserConns, tag)
						MuUserConns.Unlock()
						conn.Close()
						return
					}

					tagBuf := make([]byte, 4)
					binary.BigEndian.PutUint32(tagBuf, tag)

					lenBuf := make([]byte, 4)
					binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

					frame := append(tagBuf, lenBuf...)
					frame = append(frame, data...)
					outgo <- frame
				}
			}(tag, appConn)
		}

		WriteToApp(appConn, data)
	}
}

// WriteToServer writes frames from apps to the server
func WriteToServer(conn net.Conn, outgo chan []byte) {
	for data := range outgo {
		if _, err := conn.Write(data); err != nil {
			log.Println("Write to server failed:", err)
			return
		}
	}
}

// ReadFull helper wraps io.ReadFull
func ReadFull(conn net.Conn, buf []byte) (int, error) {
	n, err := io.ReadFull(conn, buf)
	if err != nil {
		return n, err
	}
	return n, nil
}
