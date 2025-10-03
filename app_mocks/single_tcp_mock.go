package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	// Listen on port 8080
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Println("Listening on :8080 ...")

	// Accept one connection
	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}

	fmt.Println("Connection accepted from", conn.RemoteAddr())

	// Keep the connection for 10 seconds
	time.Sleep(10 * time.Second)

	// Close the connection
	conn.Close()
	fmt.Println("Connection closed after 10 seconds")
}
