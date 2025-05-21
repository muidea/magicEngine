package main

import (
	"flag"

	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/log"
	"github.com/muidea/magicEngine/tcp"
)

type Echo struct {
	recvCount int
	recvSize  int
	sendCount int
	sendSize  int
}

func (s *Echo) OnConnect(ep tcp.Endpoint) {
	log.Infof("new connect, from:%s", ep.RemoteAddr().String())
}

func (s *Echo) OnDisConnect(ep tcp.Endpoint) {
	log.Infof("disconnect, from:%s", ep.RemoteAddr().String())
}

func (s *Echo) OnRecvData(ep tcp.Endpoint, data []byte) {
	dataSize := len(data)
	s.recvCount++
	s.recvSize += dataSize
	log.Infof("recv data from:%s, recvCount:%d, size:%d", ep.RemoteAddr().String(), s.recvCount, dataSize)
	ep.SendData(data)
}

var bindAddr = "0.0.0.0:8080"

func main() {
	flag.StringVar(&bindAddr, "bindAddr", bindAddr, "listen address")
	flag.Parse()

	echo := &Echo{}

	executePtr := execute.NewExecute(1000)
	epManaer := tcp.NewEndpointManger(echo, &executePtr)
	svr := tcp.NewServer(epManaer, &executePtr)
	svr.Run(bindAddr)
}
