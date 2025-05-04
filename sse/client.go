package sse

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
)

type Client struct {
	serverURI   string
	maxRetries  int
	retryWait   time.Duration
	lastEventID string
	syncMutex   sync.Mutex

	cancelFunc context.CancelFunc
}

type Event struct {
	ID    string
	Name  string
	Data  []byte
	Retry time.Duration
}

func NewClient(uri string, retryWait time.Duration, maxRetries int) *Client {
	return &Client{
		serverURI:  uri,
		maxRetries: maxRetries,
	}
}

func (s *Client) Close() {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

func (s *Client) Get(ctx context.Context, header url.Values, sink StreamSinker) error {
	urlVal, urlErr := url.ParseRequestURI(s.serverURI)
	if urlErr != nil {
		log.Errorf("parse url failed, err:%s", urlErr)
		return urlErr
	}

	actionFunc := func() (err error) {
		clientCtx, clientCancel := context.WithCancel(context.Background())
		defer func() {
			s.syncMutex.Lock()
			defer s.syncMutex.Unlock()
			if err != nil {
				clientCancel()
				s.cancelFunc = nil
			}
		}()

		func() {
			s.syncMutex.Lock()
			defer s.syncMutex.Unlock()

			s.cancelFunc = clientCancel
		}()

		requestVal, requestErr := http.NewRequestWithContext(clientCtx, http.MethodGet, urlVal.String(), nil)
		if requestErr != nil {
			err = requestErr
			log.Errorf("new request failed, err:%s", err)
			return
		}

		for k, v := range header {
			requestVal.Header.Set(k, v[0])
		}
		requestVal.Header.Set("Accept", sseStream)
		requestVal.Header.Set("Cache-Control", "no-cache")
		if s.lastEventID != "" {
			requestVal.Header.Set("Last-Event-ID", s.lastEventID)
		}
		responseVal, responseErr := http.DefaultClient.Do(requestVal)
		if responseErr != nil {
			err = responseErr
			log.Errorf("do request failed, err:%s", err)
			return
		}
		defer responseVal.Body.Close()

		if responseVal.StatusCode != http.StatusOK {
			err = fmt.Errorf("request failed, status:%d", responseVal.StatusCode)
			return
		}

		err = s.recvVal(clientCtx, responseVal, sink)
		return
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
					sink.OnClose()

					return err
				}

				retryCount = retryVal + 1
				continue
			}

			retryCount = 0
			return err
		}
	}
}

func (s *Client) Post(ctx context.Context, byteVal []byte, header url.Values, sink StreamSinker) error {
	urlVal, urlErr := url.ParseRequestURI(s.serverURI)
	if urlErr != nil {
		log.Errorf("parse url failed, err:%s", urlErr)
		return urlErr
	}

	actionFunc := func() (err error) {
		clientCtx, clientCancel := context.WithCancel(context.Background())
		defer func() {
			s.syncMutex.Lock()
			defer s.syncMutex.Unlock()
			if err != nil {
				clientCancel()
				s.cancelFunc = nil
			}
		}()

		func() {
			s.syncMutex.Lock()
			defer s.syncMutex.Unlock()

			s.cancelFunc = clientCancel
		}()

		byteBuff := bytes.NewBuffer(nil)
		if byteVal != nil {
			byteBuff.Write(byteVal)
		}

		requestVal, requestErr := http.NewRequestWithContext(clientCtx, http.MethodPost, urlVal.String(), byteBuff)
		if requestErr != nil {
			err = requestErr
			log.Errorf("new request failed, err:%s", err)
			return
		}
		for k, v := range header {
			requestVal.Header.Set(k, v[0])
		}
		requestVal.Header.Set("Accept", sseStream)
		requestVal.Header.Set("Cache-Control", "no-cache")
		if s.lastEventID != "" {
			requestVal.Header.Set("Last-Event-ID", s.lastEventID)
		}
		responseVal, responseErr := http.DefaultClient.Do(requestVal)
		if responseErr != nil {
			err = responseErr
			log.Errorf("do request failed, err:%s", err)
			return
		}
		defer responseVal.Body.Close()

		if responseVal.StatusCode != http.StatusOK {
			contentVal, contentErr := io.ReadAll(responseVal.Body)
			if contentErr != nil {
				err = contentErr
				log.Errorf("read content failed, err:%s", err)
				return
			}

			err = fmt.Errorf("request failed, status:%d, message:%s", responseVal.StatusCode, string(contentVal))
			return
		}

		err = s.recvVal(ctx, responseVal, sink)
		return
	}

	var retryCount int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var err error
			if err = actionFunc(); err != nil {
				log.Errorf("action failed, serverUrl:%s, err:%v", s.serverURI, err)
				retryVal, retryErr := s.handleRetry(retryCount)
				if retryErr != nil {
					log.Errorf("handle retry failed, err:%s", retryErr)
					sink.OnClose()
					return err
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
func (s *Client) recvVal(ctx context.Context, resp *http.Response, sink StreamSinker) (err error) {
	reader := bufio.NewReader(resp.Body)
	var currentEvent Event

	for {
		select {
		case <-ctx.Done():
			return
		default:
			byteVal, byteErr := reader.ReadBytes('\n')
			if byteErr != nil {
				return
			}

			byteVal = bytes.TrimSpace(byteVal)
			if len(byteVal) == 0 {
				// 空行表示事件结束
				if currentEvent.Data != nil {
					s.syncMutex.Lock()
					s.lastEventID = currentEvent.ID
					s.syncMutex.Unlock()
					sink.OnRecv(currentEvent.Name, currentEvent.Data)
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
