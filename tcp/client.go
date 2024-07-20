package tcp

import (
	"fmt"
	"net"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
)

type Client interface {
	Endpoint
	Connect(serverAddr string) error
}

type clientImpl struct {
	execute.Execute

	observer Observer
	endpoint Endpoint
}

func NewClient(ob Observer) Client {
	return &clientImpl{
		Execute:  execute.NewExecute(10),
		observer: ob,
	}
}

func (s *clientImpl) Connect(serverAddr string) (err error) {
	connVal, connErr := net.Dial("tcp", serverAddr)
	if connErr != nil {
		log.Errorf("connect %s failed, error:%s", serverAddr, connErr.Error())
		err = connErr
		return
	}
	if s.observer == nil {
		connVal.Close()
		return
	}

	implPtr := NewEndpoint(connVal, s.observer, &s.Execute)
	s.endpoint = implPtr

	s.Execute.Run(func() {
		log.Infof("connect remote server %s ok", serverAddr)
		defer implPtr.Close()
		implPtr.RecvData()
	})

	return
}

func (s *clientImpl) Close() {
	if s.endpoint != nil {
		s.endpoint.Close()
	}
}

func (s *clientImpl) SendData(data []byte) error {
	if s.endpoint == nil {
		return fmt.Errorf("illegal endpoint, must connect first")
	}

	return s.endpoint.SendData(data)
}

func (s *clientImpl) LocalAddr() net.Addr {
	if s.endpoint == nil {
		return nil
	}

	return s.endpoint.LocalAddr()
}

func (s *clientImpl) RemoteAddr() net.Addr {
	if s.endpoint == nil {
		return nil
	}

	return s.endpoint.RemoteAddr()
}
