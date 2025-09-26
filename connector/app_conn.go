package main

import (
	"fmt"
	"log"
	"net"
)

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

// WriteToApp writes data to the app connection safely
func WriteToApp(conn net.Conn, userConnId uint32, data []byte) error {
	if conn == nil {
		log.Println("No connection established to the app for user", userConnId)
		return fmt.Errorf("connection is nil")
	}
	if _, err := conn.Write(data); err != nil {
		log.Println("Write to app failed for user", userConnId, "because", err)
		return fmt.Errorf("app write failed")
	}
	return nil
}

// ReadFromApp reads data from the app connection safely
func ReadFromApp(conn net.Conn) ([]byte, bool) {
	if conn == nil {
		return nil, false
	}
	buff := make([]byte, 1024)
	n, err := conn.Read(buff)
	if err != nil {
		return nil, false
	}
	return buff[:n], true
}
