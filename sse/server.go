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

type Observer interface {
	OnClose(id string)
}

type Holder struct {
	httpResponseWriter http.ResponseWriter
	httpRequest        *http.Request
	lastActive         time.Time
	mu                 *sync.Mutex

	sseID      string
	observer   Observer
	masterFlag bool
}

func (s *Holder) OnRecv(event string, data []byte) (err error) {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()

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
	return
}

func (s *Holder) OnClose() {
	if s.observer != nil {
		s.observer.OnClose(s.sseID)
	}
}

func (s *Holder) heartbeat() (err error) {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()

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

func (s *Holder) echoSSEID() {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", sseStream)
	_, err := s.httpResponseWriter.Write(fmt.Appendf(nil, "sseID: %s\n", s.sseID))
	if err != nil {
		log.Errorf("write heartbeat failed, err:%s", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}
}

func (s *Holder) Run(taskFunc func() error) error {
	// 这里主动进行限制，已有一个Master，在进行心跳检测
	var curMasterFlag bool
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		curMasterFlag = s.masterFlag
	}()
	if curMasterFlag {
		if taskFunc == nil {
			return nil
		}

		return taskFunc()
	}

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
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

		s.echoSSEID()

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
				s.mu.Lock()
				lastActive := s.lastActive // 获取最后活动时间副本
				s.mu.Unlock()

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
	holder := NewHolder(res, req)
	holder.mu = &s.mu
	holder.sseID = pu.RandomAlphanumeric(32)
	holder.observer = s

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
