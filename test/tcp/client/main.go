package main

import (
	"flag"
	"fmt"
	"github.com/muidea/magicCommon/foundation/log"
	"github.com/muidea/magicCommon/foundation/util"
	"github.com/muidea/magicEngine/tcp"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Echo struct {
	recvCount int
	recvSize  int
	sendCount int
	sendSize  int
}

func (s *Echo) OnConnect(ep tcp.Endpoint) {
	log.Infof("connect ok, server:%s", ep.RemoteAddr().String())
}

func (s *Echo) OnDisConnect(ep tcp.Endpoint) {
	log.Infof("disconnect, server:%s", ep.RemoteAddr().String())
}

func (s *Echo) OnRecvData(ep tcp.Endpoint, data []byte) {
	dataSize := len(data)
	s.recvCount++
	s.recvSize += dataSize
	log.Infof("recv data from:%s, recvCount:%d, size:%d", ep.RemoteAddr().String(), s.recvCount, dataSize)
}

func (s *Echo) Dump() {
	msg := fmt.Sprintf("\nsendCount:%d\nsendSize:%d\nrecvCount:%d\nrecvSize:%d\n",
		s.sendCount,
		s.sendSize,
		s.recvCount,
		s.recvSize)
	log.Infof(msg)
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
		log.Errorf("connect %s failed, error:%s", serverAddr, err.Error())
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
			log.Errorf("sendData failed, error:%s", err.Error())
			continue
		}

		time.Sleep(1 * time.Second)
	}

	echo.Dump()
}
