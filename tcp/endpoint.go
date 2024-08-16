package tcp

import (
	"bufio"
	"net"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
)

type Endpoint interface {
	Close()
	SendData(data []byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Observer interface {
	OnConnect(ep Endpoint)
	OnDisConnect(ep Endpoint)
	OnRecvData(ep Endpoint, data []byte)
}

func NewEndpoint(conn net.Conn, ob Observer, executePtr *execute.Execute) *endpointImpl {
	ptr := &endpointImpl{
		connVal:    conn,
		observer:   ob,
		executePtr: executePtr,
	}

	if ob != nil {
		executePtr.Run(func() {
			ob.OnConnect(ptr)
		})
	}

	return ptr
}

const buffSize = 1024

type endpointImpl struct {
	connVal    net.Conn
	observer   Observer
	executePtr *execute.Execute
}

func (s *endpointImpl) Close() {
	s.observer = nil

	_ = s.connVal.Close()
}

func (s *endpointImpl) SendData(data []byte) (err error) {
	offSet := 0
	totalSize := len(data)
	for {
		sendSize, sendErr := s.connVal.Write(data[offSet : totalSize-offSet])
		if sendErr != nil {
			err = sendErr
			break
		}

		offSet += sendSize
		if offSet >= totalSize {
			break
		}
	}

	if err != nil && s.observer != nil {
		s.executePtr.Run(func() {
			s.observer.OnDisConnect(s)
		})
	}

	return
}

func (s *endpointImpl) LocalAddr() net.Addr {
	return s.connVal.LocalAddr()
}

func (s *endpointImpl) RemoteAddr() net.Addr {
	return s.connVal.RemoteAddr()
}

func (s *endpointImpl) RecvData() (err error) {
	reader := bufio.NewReader(s.connVal)
	buffer := make([]byte, buffSize)
	for {
		readSize, readErr := reader.Read(buffer)
		if readErr != nil {
			log.Errorf("recv data failed")
			err = readErr
			break
		}

		if s.observer != nil && readSize > 0 {
			s.observer.OnRecvData(s, buffer[:readSize])
		}
	}

	if s.observer != nil {
		s.observer.OnDisConnect(s)
	}
	return nil
}
