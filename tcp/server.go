package tcp

import (
	"net"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
)

type Server interface {
	Run(bindAddr string) error
}

type ServerSink interface {
	OnNewConnect(conn net.Conn)
}

type serverImpl struct {
	executePtr *execute.Execute
	serverSink ServerSink
}

func NewServer(sink ServerSink, executePtr *execute.Execute) Server {
	return &serverImpl{
		executePtr: executePtr,
		serverSink: sink,
	}
}

func (s *serverImpl) Run(bindAddr string) (err error) {
	listenerVal, listenerErr := net.Listen("tcp", bindAddr)
	if listenerErr != nil {
		log.Errorf("listen %s failed, error:%s", bindAddr, listenerErr.Error())
		err = listenerErr
		return
	}
	defer listenerVal.Close()

	log.Infof("TCP Server started. Listening on %s", bindAddr)
	for {
		connVal, connErr := listenerVal.Accept()
		if connErr != nil {
			log.Errorf("accept new connect failed, error:%s", connErr.Error())
			continue
		}

		log.Infof("accept new connect, from:%s", connVal.RemoteAddr().String())
		s.executePtr.Run(func() {
			if s.serverSink == nil {
				_ = connVal.Close()
				return
			}

			s.serverSink.OnNewConnect(connVal)
		})
	}
}
