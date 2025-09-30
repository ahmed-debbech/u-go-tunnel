package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type Frame struct {
	ConnId  uint32
	Length  uint32 //Data length
	AppPort uint16
	Data    []byte
}

func ParseFrame(conn net.Conn) (Frame, error) {

	frame := Frame{}

	idBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, idBuf); err != nil {
		log.Println("Read ConnId failed:", err)
		return Frame{}, fmt.Errorf("Could not read ConnId")
	}
	connId := binary.BigEndian.Uint32(idBuf)
	frame.ConnId = connId

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		log.Println(connId, "Read Length failed:", err)
		return Frame{}, fmt.Errorf("Could not read Length")
	}
	length := binary.BigEndian.Uint32(lenBuf)
	frame.Length = length

	appPortBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, appPortBuf); err != nil {
		log.Println(connId, "Read AppPort failed:", err)
		return Frame{}, fmt.Errorf("Could not read AppPort")
	}
	appPort := binary.BigEndian.Uint16(appPortBuf)
	frame.AppPort = appPort

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		log.Println(connId, "Read DATA failed:", err)
		return Frame{}, fmt.Errorf("Could not read Data")
	}
	frame.Data = data

	return frame, nil
}

func ConstructFrame(tag uint32, appPort uint16, data []byte) Frame {

	frame := Frame{
		ConnId:  tag,
		Length:  uint32(len(data)),
		Data:    data,
		AppPort: appPort,
	}

	return frame
}

func SerializeFrame(frame Frame) []byte {

	tagBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(tagBuf, frame.ConnId)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, frame.Length)

	appPortBuf := make([]byte, 4)
	binary.BigEndian.PutUint16(appPortBuf, frame.AppPort)

	fr := append(tagBuf, lenBuf...)
	fr = append(fr, appPortBuf...)
	fr = append(fr, frame.Data...)

	return fr
}
