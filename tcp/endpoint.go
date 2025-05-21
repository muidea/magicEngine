package tcp

import (
	"bufio"
	"net"
	"sync"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
)

type Endpoint interface {
	Close()
	SendData(data []byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	String() string
}

type Observer interface {
	OnConnect(ep Endpoint)
	OnDisConnect(ep Endpoint)
	OnRecvData(ep Endpoint, data []byte)
}

type SimpleEndpointManger struct {
	observer   Observer
	executePtr *execute.Execute

	endpointMap sync.Map
}

func NewEndpointManger(observer Observer, executePtr *execute.Execute) *SimpleEndpointManger {
	ptr := &SimpleEndpointManger{
		observer:   observer,
		executePtr: executePtr,
	}

	return ptr
}

func (s *SimpleEndpointManger) OnNewConnect(conn net.Conn) {
	if s.observer == nil {
		_ = conn.Close()
		return
	}

	endpointPtr := newEndpoint(conn, s)
	defer endpointPtr.Close()
	_ = endpointPtr.RecvData()

}

func (s *SimpleEndpointManger) OnConnect(ep Endpoint) {
	if s.observer == nil {
		return
	}

	s.endpointMap.Store(ep.String(), ep)
	s.executePtr.Run(func() {
		s.observer.OnConnect(ep)
	})
}

func (s *SimpleEndpointManger) OnDisConnect(ep Endpoint) {
	if s.observer == nil {
		return
	}

	s.endpointMap.Delete(ep.String())
	s.executePtr.Run(func() {
		s.observer.OnDisConnect(ep)
	})
}

func (s *SimpleEndpointManger) OnRecvData(ep Endpoint, data []byte) {
	if s.observer == nil {
		return
	}

	s.executePtr.Run(func() {
		s.observer.OnRecvData(ep, data)
	})
}

func newEndpoint(conn net.Conn, ob Observer) *endpointImpl {
	ptr := &endpointImpl{
		connVal:  conn,
		observer: ob,
	}

	if ob != nil {
		ob.OnConnect(ptr)
	}

	return ptr
}

const buffSize = 1024

type endpointImpl struct {
	connVal  net.Conn
	observer Observer
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
		s.observer.OnDisConnect(s)
	}

	return
}

func (s *endpointImpl) LocalAddr() net.Addr {
	return s.connVal.LocalAddr()
}

func (s *endpointImpl) RemoteAddr() net.Addr {
	return s.connVal.RemoteAddr()
}

func (s *endpointImpl) String() string {
	return s.connVal.LocalAddr().String() + "->" + s.connVal.RemoteAddr().String()
}

func (s *endpointImpl) RecvData() (err error) {
	reader := bufio.NewReader(s.connVal)
	buffer := make([]byte, buffSize)
	for {
		readSize, readErr := reader.Read(buffer)
		if readErr != nil {
			log.Errorf("recv data failed, error:%s", readErr.Error())
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
