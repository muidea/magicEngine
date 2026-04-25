package tcp

import (
	"net"

	"log/slog"

	"github.com/muidea/magicCommon/execute"
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
		slog.Error("listen failed", "addr", bindAddr, "err", listenerErr)
		err = listenerErr
		return
	}
	defer func() {
		_ = listenerVal.Close()
	}()

	slog.Info("TCP server started", "addr", bindAddr)
	for {
		connVal, connErr := listenerVal.Accept()
		if connErr != nil {
			slog.Error("accept new connection failed", "err", connErr)
			continue
		}

		slog.Info("accepted new connection", "from", connVal.RemoteAddr().String())
		s.executePtr.Run(func() {
			if s.serverSink == nil {
				_ = connVal.Close()
				return
			}

			s.serverSink.OnNewConnect(connVal)
		})
	}
}
