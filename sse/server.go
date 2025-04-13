package sse

import (
	"net/http"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
)

const (
	timerInterval = 1 * time.Second
	timerTimeout  = 3 * time.Second
)

func IsSSE(req *http.Request) bool {
	return req.Header.Get("Accept") == "text/event-stream"
}

type Holder struct {
	httpResponseWriter http.ResponseWriter
	httpRequest        *http.Request
	lastActive         time.Time
	mu                 sync.Mutex
}

func (s *Holder) OnRecv(event string, data []byte) (err error) {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", "text/event-stream")
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

}

func (s *Holder) heartbeat() (err error) {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()

	s.httpResponseWriter.Header().Add("Content-Type", "text/event-stream")
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

func (s *Holder) Run(taskFunc func() error) error {
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
	}
}
