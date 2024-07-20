package tcp

import (
	"net"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
)

type Server interface {
	Run(bindAddr string) error
}

type serverImpl struct {
	execute.Execute

	observer Observer
}

func NewServer(ob Observer, maxConnSize int) Server {
	return &serverImpl{
		Execute:  execute.NewExecute(maxConnSize),
		observer: ob,
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

		s.Execute.Run(func() {
			log.Infof("accept new connect, from:%s", connVal.RemoteAddr().String())
			if s.observer == nil {
				connVal.Close()
				return
			}

			endpoint := NewEndpoint(connVal, s.observer, &s.Execute)
			defer endpoint.Close()
			endpoint.RecvData()
		})
	}

	return
}
