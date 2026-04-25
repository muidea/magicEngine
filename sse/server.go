package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"log/slog"

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

type holderSink interface {
	OnClose(id string)
}

type StreamSink struct {
}

func (s *StreamSink) String() string {
	return "StreamSink"
}

type StreamSinker interface {
	OnClose()
	OnRecv(event string, content []byte)
}

type Holder struct {
	httpResponseWriter http.ResponseWriter
	httpRequest        *http.Request
	lastActive         time.Time
	syncMutexPtr       *sync.Mutex

	sseID      string
	sinker     holderSink
	masterFlag bool
}

func (s *Holder) OnRecv(event string, data []byte) {
	var err error
	defer func() {
		if err != nil && s.sinker != nil {
			s.sinker.OnClose(s.sseID)
		}
	}()
	s.syncMutexPtr.Lock()
	s.lastActive = time.Now()
	s.syncMutexPtr.Unlock()

	s.httpResponseWriter.Header().Set("Content-Type", sseStream)
	if event != "" {
		_, err = s.httpResponseWriter.Write([]byte("event: " + event + "\n"))
		if err != nil {
			slog.Error("write event failed", "err", err)
			return
		}
	}
	_, err = s.httpResponseWriter.Write([]byte("data: " + string(data) + "\n\n"))
	if err != nil {
		slog.Error("write data failed", "err", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}
}

func (s *Holder) OnClose() {
	if s.sinker != nil {
		s.sinker.OnClose(s.sseID)
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

	s.httpResponseWriter.Header().Set("Content-Type", sseStream)
	_, err = s.httpResponseWriter.Write([]byte(": ping\n\n"))
	if err != nil {
		slog.Error("write heartbeat failed", "err", err)
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

	s.httpResponseWriter.Header().Set("Content-Type", sseStream)
	_, err = s.httpResponseWriter.Write(fmt.Appendf(nil, "event: sseID\ndata: %s\n\n", s.sseID))
	if err != nil {
		slog.Error("write heartbeat failed", "err", err)
		return
	}

	flusherVal, flusherOK := s.httpResponseWriter.(http.Flusher)
	if flusherOK {
		flusherVal.Flush()
	}

	return
}

func (s *Holder) Run(taskFunc func() error) error {
	if taskFunc == nil {
		return nil
	}

	// 这里主动进行限制，已有一个Master，在进行心跳检测
	var curMasterFlag bool
	func() {
		s.syncMutexPtr.Lock()
		defer s.syncMutexPtr.Unlock()
		curMasterFlag = s.masterFlag
	}()
	if curMasterFlag {
		return taskFunc()
	}

	func() {
		s.syncMutexPtr.Lock()
		defer s.syncMutexPtr.Unlock()
		s.masterFlag = true
	}()

	done := make(chan struct{})
	var wg sync.WaitGroup
	var runErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		runErr = taskFunc()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(timerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				s.syncMutexPtr.Lock()
				lastActive := s.lastActive
				s.syncMutexPtr.Unlock()

				if time.Since(lastActive) > timerTimeout {
					_ = s.heartbeat()
				}
			}
		}
	}()

	wg.Wait()
	return runErr
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
