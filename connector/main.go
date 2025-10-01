package main

import (
	"encoding/binary"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	Config      = "./ports.conf"
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

		ports, err := ParseConfig(Config)
		if err != nil {
			log.Fatal()
		}

		meta := make([]byte, 0)
		for _, v := range ports {
			l := make([]byte, 2)
			binary.BigEndian.PutUint16(l, v)
			meta = append(meta, l...)
		}

		configLen := make([]byte, 4)
		binary.BigEndian.PutUint32(configLen, uint32(len(meta)))

		if _, err := conn.Write(append(configLen, meta...)); err != nil {
			log.Println("COuld not send config data to server")
			log.Fatal()
		}
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

		//if it is closing frame then close connection to app
		if frame.Length == 0 {
			log.Println(frame.ConnId, "user requested to close their connection to the app")
			if ok {
				appConn.Close()
				log.Println(frame.ConnId, "user requested to close their connection to the app")
				MuUserConns.Lock()
				delete(UserConns, frame.ConnId)
				MuUserConns.Unlock()
			}
			continue
		}

		if !ok || appConn == nil {

			appport := strconv.Itoa(int(frame.AppPort))
			appConn = ConnectToApp(frame.ConnId, "localhost", appport)
			if appConn == nil {
				log.Println(frame.ConnId, "Cannot connect to app, skipping frame")
				frame := ConstructFrame(frame.ConnId, frame.AppPort, []byte{})
				MuUserConns.Lock()
				delete(UserConns, frame.ConnId)
				MuUserConns.Unlock()

				outgo <- frame
				continue
			}

			MuUserConns.Lock()
			UserConns[frame.ConnId] = appConn
			MuUserConns.Unlock()

			go func() {
				for {
					data, ok := ReadFromApp(appConn)
					if !ok {
						log.Println("App closed connection for user ", frame.ConnId)
						MuUserConns.Lock()
						delete(UserConns, frame.ConnId)
						MuUserConns.Unlock()
						appConn.Close()

						frame := ConstructFrame(frame.ConnId, frame.AppPort, []byte{})
						outgo <- frame
						return
					}

					frame := ConstructFrame(frame.ConnId, frame.AppPort, data)
					outgo <- frame
				}
			}()
		}

		if err := WriteToApp(appConn, frame.ConnId, frame.Data); err != nil {
			log.Println("App closed connection for user ", frame.ConnId)
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
