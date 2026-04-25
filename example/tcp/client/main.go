package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/muidea/magicCommon/foundation/util"
	"github.com/muidea/magicEngine/tcp"
)

type Echo struct {
	recvCount int
	recvSize  int
	sendCount int
	sendSize  int
}

func (s *Echo) OnConnect(ep tcp.Endpoint) {
	slog.Info("connect ok", "server", ep.RemoteAddr().String())
}

func (s *Echo) OnDisConnect(ep tcp.Endpoint) {
	slog.Info("disconnect", "server", ep.RemoteAddr().String())
}

func (s *Echo) OnRecvData(ep tcp.Endpoint, data []byte) {
	dataSize := len(data)
	s.recvCount++
	s.recvSize += dataSize
	slog.Info("recv data", "from", ep.RemoteAddr().String(), "recvCount", s.recvCount, "size", dataSize)
}

func (s *Echo) Dump() {
	msg := fmt.Sprintf("\nsendCount:%d\nsendSize:%d\nrecvCount:%d\nrecvSize:%d\n",
		s.sendCount,
		s.sendSize,
		s.recvCount,
		s.recvSize)
	slog.Info("dump stats", "msg", msg)
}

var serverAddr = "127.0.0.1:8080"

func main() {
	flag.StringVar(&serverAddr, "serverAddr", serverAddr, "server address")
	flag.Parse()

	echo := &Echo{}

	// 创建一个通道来接收信号
	signalCh := make(chan os.Signal, 1)

	// 将 SIGINT（Ctrl+C）和 SIGTERM 信号发送到通道
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	runningFlag := true
	go func() {
		<-signalCh

		runningFlag = false
		fmt.Println("\nCtrl+C pressed. Cleaning up...")
	}()

	clnt := tcp.NewClient(echo)
	err := clnt.Connect(serverAddr)
	if err != nil {
		slog.Error("connect failed", "server", serverAddr, "err", err)
		return
	}
	defer clnt.Close()

	for runningFlag {
		sendMsg := util.RandomString(120)
		dataSize := len(sendMsg)
		echo.sendCount++
		echo.sendSize += dataSize

		err = clnt.SendData([]byte(sendMsg))
		if err != nil {
			slog.Error("sendData failed", "err", err)
			continue
		}

		time.Sleep(1 * time.Second)
	}

	echo.Dump()
}
