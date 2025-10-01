package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
)

var (
	ConnectorConnection   = make(map[uint32]Tunnel) //Every Connection to a linked Connector by its id
	MuConnectorConnection sync.RWMutex

	ExposedPorts = make(map[uint16]uint32) //exposed ports on server to alternative connectors ids that handles it

)

func ReadConnectorConfig(conn net.Conn, id uint32) ([]uint16, error) {
	configLen := make([]byte, 4)
	if _, err := io.ReadFull(conn, configLen); err != nil {
		log.Println("Read LENCONFIG failed:", err)
		return nil, fmt.Errorf("Read LENCONFIG failed:", err)
	}
	configLenN := binary.BigEndian.Uint32(configLen)
	data := make([]byte, configLenN)
	if _, err := io.ReadFull(conn, data); err != nil {
		log.Println("Read Config data failed:", err)
		return nil, fmt.Errorf("Read Config data failed:", err)
	}

	ports := make([]uint16, 0)
	for i := 0; i <= len(data)-2; i += 2 {
		ff := make([]byte, 0)
		ff = append(ff, data[i])
		ff = append(ff, data[i+1])
		c := binary.BigEndian.Uint16(ff)
		ports = append(ports, c)
	}

	for _, v := range ports {
		ExposedPorts[v] = id
	}
	return ports, nil
}

func CleanUpConnector(ConnectorConnection map[uint32]Tunnel, id uint32) {
	MuConnectorConnection.RLock()
	log.Println("could not get port config for connector", id)
	ConnectorConnection[id].ConnectorConn.Close()
	close(ConnectorConnection[id].FromUserToConnector)
	delete(ConnectorConnection, id)
	MuConnectorConnection.RUnlock()

	for k, v := range ExposedPorts {
		if v == id {
			delete(ExposedPorts, k)
		}
	}
}

func GenerateNewConnectorId(conn net.Conn, fromUserToConnector chan Frame) uint32 {
	var id uint32
	for {
		id = rand.Uint32()
		MuConnectorConnection.Lock()
		if _, exists := ConnectorConnection[id]; !exists {
			tun := Tunnel{
				ConnectorConn:       conn,
				FromUserToConnector: fromUserToConnector,
			}
			ConnectorConnection[id] = tun
			MuConnectorConnection.Unlock()
			break
		}
		MuConnectorConnection.Unlock()
	}
	return id
}
