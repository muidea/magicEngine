package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
	pu "github.com/muidea/magicCommon/foundation/util"
)

const (
	sseID     = "X-SSE-ID"
	sseStream = "text/event-stream"
)

const (
	timerInterval = 1 * time.Second
	timerTimeout  = 3 * time.Second
)

func IsSSE(req *http.Request) bool {
	return req.Header.Get("Accept") == sseStream
}

type holderSinker interface {
	OnClose(id string)
}

type Writer interface {
	SendData(event string, data []byte)
}

type Holder struct {
	httpResponseWriter http.ResponseWriter
	httpRequest        *http.Request
	lastActive         time.Time
	syncMutexPtr       *sync.Mutex

	sseID      string
	sinker     holderSinker
	masterFlag bool
}

func (s *Holder) SendData(event string, data []byte) {
	var err error
	defer func() {
		if err != nil && s.sinker != nil {
			s.sinker.OnClose(s.sseID)
		}
	}()
	s.syncMutexPtr.Lock()
	s.lastActive = time.Now()
	s.syncMutexPtr.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", sseStream)
	if event != "" {
		_, err = s.httpResponseWriter.Write([]byte("event: " + event + "\n"))
		if err != nil {
			log.Errorf("write event failed, err:%s", err)
			return
		}
	}
	_, err = s.httpResponseWriter.Write([]byte("data: " + string(data) + "\n"))
	if err != nil {
		log.Errorf("write data failed, err:%s", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}
}

func (s *Holder) heartbeat() (err error) {
	defer func() {
		if err != nil && s.sinker != nil {
			s.sinker.OnClose(s.sseID)
		}
	}()

	s.syncMutexPtr.Lock()
	s.lastActive = time.Now()
	s.syncMutexPtr.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", sseStream)
	_, err = s.httpResponseWriter.Write([]byte(": ping\n"))
	if err != nil {
		log.Errorf("write heartbeat failed, err:%s", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}
	return
}

func (s *Holder) EchoSSEID() (err error) {
	defer func() {
		if err != nil && s.sinker != nil {
			s.sinker.OnClose(s.sseID)
		}
	}()

	if s.syncMutexPtr == nil {
		return
	}

	s.syncMutexPtr.Lock()
	s.lastActive = time.Now()
	s.syncMutexPtr.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", sseStream)
	_, err = s.httpResponseWriter.Write(fmt.Appendf(nil, "event: sseID\ndata: %s\n\n", s.sseID))
	if err != nil {
		log.Errorf("write heartbeat failed, err:%s", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}

	return
}

func (s *Holder) Run(taskFunc func() error) error {
	// 这里主动进行限制，已有一个Master，在进行心跳检测
	var curMasterFlag bool
	func() {
		s.syncMutexPtr.Lock()
		defer s.syncMutexPtr.Unlock()
		curMasterFlag = s.masterFlag
	}()
	if curMasterFlag {
		if taskFunc == nil {
			return nil
		}

		return taskFunc()
	}

	func() {
		s.syncMutexPtr.Lock()
		defer s.syncMutexPtr.Unlock()
		s.masterFlag = true
	}()

	wg := &sync.WaitGroup{}

	taskOK := false
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			taskOK = true
		}()

		err := taskFunc()
		if err != nil {
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(timerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.syncMutexPtr.Lock()
				lastActive := s.lastActive // 获取最后活动时间副本
				s.syncMutexPtr.Unlock()

				if time.Since(lastActive) > timerTimeout {
					s.heartbeat()
				}
			default:
				if taskOK {
					return
				}
			}
		}
	}()

	wg.Wait()
	return nil
}

func NewHolder(res http.ResponseWriter, req *http.Request) *Holder {
	return &Holder{
		httpResponseWriter: res,
		httpRequest:        req,
		masterFlag:         false,
		syncMutexPtr:       &sync.Mutex{},
		sseID:              pu.RandomAlphanumeric(32),
	}
}

type HolderRegistry struct {
	holderMap sync.Map
	mu        sync.Mutex
}

func CreateHolderRegistry() *HolderRegistry {
	ptr := &HolderRegistry{}

	return ptr
}

func (s *HolderRegistry) NewHolder(res http.ResponseWriter, req *http.Request) *Holder {
	holder := &Holder{
		httpResponseWriter: res,
		httpRequest:        req,
		masterFlag:         false,
		syncMutexPtr:       &s.mu,
		sseID:              pu.RandomAlphanumeric(32),
		sinker:             s,
	}

	s.holderMap.Store(holder.sseID, holder)
	return holder
}

func (s *HolderRegistry) GetHolder(res http.ResponseWriter, req *http.Request) *Holder {
	sseID := req.Header.Get(sseID)
	if sseID == "" {
		return nil
	}

	holderVal, holderOK := s.holderMap.Load(sseID)
	if holderOK {
		holder := holderVal.(*Holder)
		return holder
	}

	return nil
}

func (s *HolderRegistry) OnClose(id string) {
	s.holderMap.Delete(id)
}
