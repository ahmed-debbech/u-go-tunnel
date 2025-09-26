package main

import (
	"log"
	"net"
)

// WriteToApp writes data to the app connection safely
func WriteToApp(conn net.Conn, data []byte) {
	if conn == nil {
		return
	}
	if _, err := conn.Write(data); err != nil {
		log.Println("Write to app failed:", err)
		conn.Close()
	}
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
