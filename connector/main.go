package main

import (
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

	outgoingReq := make(chan Frame, 1)

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

// ReadFromServer reads frames from server and forwards to apps
func ReadFromServer(conn net.Conn, outgo chan Frame) {
	for {

		frame, err := ParseFrame(conn)
		if err != nil {
			conn.Close()
			return
		}

		MuUserConns.Lock()
		appConn, ok := UserConns[frame.ConnId]
		MuUserConns.Unlock()

		if !ok || appConn == nil {

			appConn = ConnectToApp(frame.ConnId, "80")
			if appConn == nil {
				log.Println(frame.ConnId, "Cannot connect to app, skipping frame")
				continue
			}

			MuUserConns.Lock()
			UserConns[frame.ConnId] = appConn
			MuUserConns.Unlock()

			go func() {
				for {
					data, ok := ReadFromApp(appConn)
					if !ok {
						log.Println("App connection closed for user ", frame.ConnId)
						MuUserConns.Lock()
						delete(UserConns, frame.ConnId)
						MuUserConns.Unlock()
						appConn.Close()
						return
					}

					frame := ConstructFrame(frame.ConnId, data)

					outgo <- frame
				}
			}()
		}

		//if it is closing frame then close connection to app
		if frame.Length == 0 {
			log.Println("user requested to close their connection to the app")
			appConn.Close()
			MuUserConns.Lock()
			delete(UserConns, frame.ConnId)
			MuUserConns.Unlock()
			continue
		}

		if err := WriteToApp(appConn, frame.ConnId, frame.Data); err != nil {
			log.Println("App connection closed for user ", frame.ConnId)
			MuUserConns.Lock()
			delete(UserConns, frame.ConnId)
			MuUserConns.Unlock()
			appConn.Close()
		}
	}
}

// WriteToServer writes frames from apps to the server
func WriteToServer(conn net.Conn, outgo chan Frame) {
	for frame := range outgo {
		data := SerializeFrame(frame)
		if _, err := conn.Write(data); err != nil {
			log.Println("Write to server failed:", err)
			conn.Close()
			return
		}
	}
}
