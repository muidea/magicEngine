package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
)

type Sinker interface {
	Sink(event string, data []byte) error
}

type Client struct {
	serverURI   string
	maxRetries  int
	retryWait   time.Duration
	lastEventID string
	mu          sync.Mutex
	sink        Sinker
}

type Event struct {
	ID    string
	Name  string
	Data  []byte
	Retry time.Duration
}

func NewClient(uri string, retryWait time.Duration, maxRetries int, sink Sinker) *Client {
	return &Client{
		serverURI:  uri,
		maxRetries: maxRetries,
		sink:       sink,
	}
}

func (s *Client) Get(ctx context.Context, header url.Values) error {
	urlVal, urlErr := url.ParseRequestURI(s.serverURI)
	if urlErr != nil {
		log.Errorf("parse url failed, err:%s", urlErr)
		return urlErr
	}

	actionFunc := func() error {
		requestVal, requestErr := http.NewRequest(http.MethodGet, urlVal.String(), nil)
		if requestErr != nil {
			log.Errorf("new request failed, err:%s", requestErr)
			return requestErr
		}
		for k, v := range header {
			requestVal.Header.Set(k, v[0])
		}
		requestVal.Header.Set("Accept", "text/event-stream")
		requestVal.Header.Set("Cache-Control", "no-cache")
		if s.lastEventID != "" {
			requestVal.Header.Set("Last-Event-ID", s.lastEventID)
		}
		responseVal, responseErr := http.DefaultClient.Do(requestVal)
		if responseErr != nil {
			log.Errorf("do request failed, err:%s", responseErr)
			return responseErr
		}
		defer responseVal.Body.Close()

		if responseVal.StatusCode != http.StatusOK {
			return fmt.Errorf("request failed, status:%d", responseVal.StatusCode)
		}

		return s.recvVal(ctx, responseVal)
	}

	var retryCount int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var err error
			if err = actionFunc(); err != nil {
				retryVal, retryErr := s.handleRetry(retryCount)
				if retryErr != nil {
					log.Errorf("handle retry failed, err:%s", retryErr)
					return retryErr
				}

				retryCount = retryVal + 1
				continue
			}

			retryCount = 0
			return err
		}
	}
}

func (s *Client) Post(ctx context.Context, param any, header url.Values) error {
	urlVal, urlErr := url.ParseRequestURI(s.serverURI)
	if urlErr != nil {
		log.Errorf("parse url failed, err:%s", urlErr)
		return urlErr
	}
	var byteVal []byte
	var byteErr error
	if param != nil {
		byteVal, byteErr = json.Marshal(param)
		if byteErr != nil {
			log.Errorf("marshal param failed, err:%s", byteErr.Error())
			return byteErr
		}
	}

	actionFunc := func() error {
		byteBuff := bytes.NewBuffer(nil)
		if byteVal != nil {
			byteBuff.Write(byteVal)
		}

		requestVal, requestErr := http.NewRequest(http.MethodPost, urlVal.String(), byteBuff)
		if requestErr != nil {
			log.Errorf("new request failed, err:%s", requestErr)
			return requestErr
		}
		for k, v := range header {
			requestVal.Header.Set(k, v[0])
		}
		requestVal.Header.Set("Accept", "text/event-stream")
		requestVal.Header.Set("Cache-Control", "no-cache")
		if s.lastEventID != "" {
			requestVal.Header.Set("Last-Event-ID", s.lastEventID)
		}
		responseVal, responseErr := http.DefaultClient.Do(requestVal)
		if responseErr != nil {
			log.Errorf("do request failed, err:%s", responseErr)
			return responseErr
		}
		defer responseVal.Body.Close()

		if responseVal.StatusCode != http.StatusOK {
			contentVal, contentErr := io.ReadAll(responseVal.Body)
			if contentErr != nil {
				log.Errorf("read content failed, err:%s", contentErr)
				return contentErr
			}
			return fmt.Errorf("request failed, status:%d, message:%s", responseVal.StatusCode, string(contentVal))
		}

		return s.recvVal(ctx, responseVal)
	}

	var retryCount int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var err error
			if err = actionFunc(); err != nil {
				log.Errorf("action failed, err:%v", err)
				retryVal, retryErr := s.handleRetry(retryCount)
				if retryErr != nil {
					log.Errorf("handle retry failed, err:%s", retryErr)
					return retryErr
				}

				retryCount = retryVal + 1
				continue
			}

			retryCount = 0
			return err
		}
	}
}

func (s *Client) Put() {
	panic("not implemented")
}

func (s *Client) Delete() {
	panic("not implemented")
}

func (s *Client) Patch() {
	panic("not implemented")
}
func (s *Client) Options() {
	panic("not implemented")
}

func (s *Client) Head() {
	panic("not implemented")
}

/*
TODO 目前不确定Server在回Event时会不会不同类型的Event混着发送
// 当前的逻辑按照不会处理。后续需要确认
*/
func (s *Client) recvVal(ctx context.Context, resp *http.Response) (err error) {
	reader := bufio.NewReader(resp.Body)
	var currentEvent Event

	for {
		select {
		case <-ctx.Done():
			return
		default:
			byteVal, byteErr := reader.ReadBytes('\n')
			if byteErr != nil {
				if byteErr != io.EOF {
					log.Errorf("read body failed, err:%s", byteErr)
				}
				return
			}

			byteVal = bytes.TrimSpace(byteVal)
			if len(byteVal) == 0 {
				// 空行表示事件结束
				if currentEvent.Data != nil {
					s.mu.Lock()
					s.lastEventID = currentEvent.ID
					s.mu.Unlock()
					s.sink.Sink(currentEvent.Name, currentEvent.Data)
				}
				currentEvent = Event{}
				continue
			}

			switch {
			case bytes.HasPrefix(byteVal, []byte("event:")):
				currentEvent.Name = string(bytes.TrimPrefix(byteVal, []byte("event:")))
			case bytes.HasPrefix(byteVal, []byte("id:")):
				currentEvent.ID = string(bytes.TrimPrefix(byteVal, []byte("id:")))
			case bytes.HasPrefix(byteVal, []byte("data:")):
				currentEvent.Data = append(currentEvent.Data,
					bytes.TrimPrefix(byteVal, []byte("data:"))...)
				currentEvent.Data = append(currentEvent.Data, '\n')
			case bytes.HasPrefix(byteVal, []byte("retry:")):
				if retry, err := time.ParseDuration(string(bytes.TrimPrefix(byteVal, []byte("retry:")))); err == nil {
					s.retryWait = retry
				}
			}
		}
	}
}

func (s *Client) handleRetry(retryCount int) (ret int, err error) {
	if retryCount >= s.maxRetries {
		err = fmt.Errorf("max retries exceeded")
		return
	}

	waitTime := s.retryWait * (1 << retryCount)
	time.Sleep(waitTime)
	return retryCount + 1, nil
}
