package tcp

import (
	"context"
	"errors"
	"net"
	"sync"

	"log/slog"

	"github.com/muidea/magicCommon/execute"
)

type Server interface {
	Run(bindAddr string) error
	Shutdown(ctx context.Context) error
}

type ServerSink interface {
	OnNewConnect(conn net.Conn)
}

type serverImpl struct {
	executePtr *execute.Execute
	serverSink ServerSink
	listenerMu sync.Mutex
	listener   net.Listener
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
	s.listenerMu.Lock()
	s.listener = listenerVal
	s.listenerMu.Unlock()
	defer func() {
		_ = listenerVal.Close()
		s.listenerMu.Lock()
		if s.listener == listenerVal {
			s.listener = nil
		}
		s.listenerMu.Unlock()
	}()

	slog.Info("TCP server started", "addr", bindAddr)
	for {
		connVal, connErr := listenerVal.Accept()
		if connErr != nil {
			if errors.Is(connErr, net.ErrClosed) {
				return nil
			}
			slog.Error("accept new connection failed", "err", connErr)
			return connErr
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

func (s *serverImpl) Shutdown(_ context.Context) error {
	s.listenerMu.Lock()
	listener := s.listener
	s.listener = nil
	s.listenerMu.Unlock()
	if listener == nil {
		return nil
	}
	return listener.Close()
}
