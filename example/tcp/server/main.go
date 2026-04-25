package main

import (
	"flag"

	"log/slog"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicEngine/tcp"
)

type Echo struct {
	recvCount int
	recvSize  int
}

func (s *Echo) OnConnect(ep tcp.Endpoint) {
	slog.Info("new connect", "from", ep.RemoteAddr().String())
}

func (s *Echo) OnDisConnect(ep tcp.Endpoint) {
	slog.Info("disconnect", "from", ep.RemoteAddr().String())
}

func (s *Echo) OnRecvData(ep tcp.Endpoint, data []byte) {
	dataSize := len(data)
	s.recvCount++
	s.recvSize += dataSize
	slog.Info("recv data", "from", ep.RemoteAddr().String(), "recvCount", s.recvCount, "size", dataSize)
	_ = ep.SendData(data)
}

var bindAddr = "0.0.0.0:8080"

func main() {
	flag.StringVar(&bindAddr, "bindAddr", bindAddr, "listen address")
	flag.Parse()

	echo := &Echo{}

	executePtr := execute.NewExecute(1000)
	epManaer := tcp.NewEndpointManger(echo, &executePtr)
	svr := tcp.NewServer(epManaer, &executePtr)
	_ = svr.Run(bindAddr)
}
