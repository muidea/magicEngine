package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// SSEClient 配置结构体
type SSEConfig struct {
	ServerURL    string
	RetryWait    time.Duration // 重试等待时间
	MaxRetries   int           // 最大重试次数
	CustomParams url.Values    // 动态查询参数
}

// SSE 客户端结构体
type SSEClient struct {
	config      *SSEConfig
	httpClient  *http.Client
	eventChan   chan Event
	lastEventID string
	mu          sync.Mutex
}

// 事件数据结构
type Event struct {
	ID    string
	Type  string
	Data  []byte
	Retry time.Duration
}

func NewSSEClient(config *SSEConfig) *SSEClient {
	return &SSEClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		eventChan:  make(chan Event, 100),
	}
}

// 核心连接方法
func (c *SSEClient) Connect(ctx context.Context) {
	var retryCount int

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 动态构建带参数的URL
			u, _ := url.Parse(c.config.ServerURL)
			u.RawQuery = c.config.CustomParams.Encode()

			req, _ := http.NewRequest("GET", u.String(), nil)
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")

			// 设置断点续传ID
			c.mu.Lock()
			if c.lastEventID != "" {
				req.Header.Set("Last-Event-ID", c.lastEventID)
			}
			c.mu.Unlock()

			resp, err := c.httpClient.Do(req)
			if err != nil {
				log.Printf("连接失败: %v", err)
				retryCount = c.handleRetry(retryCount)
				continue
			}

			if resp.StatusCode != 200 {
				resp.Body.Close()
				retryCount = c.handleRetry(retryCount)
				continue
			}

			retryCount = 0 // 重置重试计数器
			c.processStream(ctx, resp)
			resp.Body.Close()
		}
	}
}

// 处理事件流
func (c *SSEClient) processStream(ctx context.Context, resp *http.Response) {
	reader := bufio.NewReader(resp.Body)
	var currentEvent Event

	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("读取错误: %v", err)
				}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				// 空行表示事件结束
				if currentEvent.Data != nil {
					c.mu.Lock()
					c.lastEventID = currentEvent.ID
					c.mu.Unlock()
					c.eventChan <- currentEvent
				}
				currentEvent = Event{}
				continue
			}

			switch {
			case bytes.HasPrefix(line, []byte("id:")):
				currentEvent.ID = string(bytes.TrimPrefix(line, []byte("id:")))
			case bytes.HasPrefix(line, []byte("data:")):
				currentEvent.Data = append(currentEvent.Data,
					bytes.TrimPrefix(line, []byte("data:"))...)
				currentEvent.Data = append(currentEvent.Data, '\n')
			case bytes.HasPrefix(line, []byte("retry:")):
				if retry, err := time.ParseDuration(string(bytes.TrimPrefix(line, []byte("retry:")))); err == nil {
					c.config.RetryWait = retry
				}
			}
		}
	}
}

// 处理重试逻辑（指数退避）
func (c *SSEClient) handleRetry(retryCount int) int {
	if retryCount >= c.config.MaxRetries {
		log.Fatal("达到最大重试次数")
	}

	waitTime := c.config.RetryWait * (1 << retryCount)
	log.Printf("等待 %v 后重试...", waitTime)
	time.Sleep(waitTime)
	return retryCount + 1
}

// 示例使用
func main() {
	config := &SSEConfig{
		ServerURL:  "http://localhost:8080/events",
		RetryWait:  1 * time.Second,
		MaxRetries: 5,
		CustomParams: url.Values{
			"category": []string{"initial"},
		},
	}

	client := NewSSEClient(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动事件处理器
	go func() {
		for event := range client.eventChan {
			fmt.Printf("收到事件 [%s]: %s\n", event.ID, event.Data)

			// 动态修改查询参数示例
			if bytes.Contains(event.Data, []byte("change_category")) {
				client.config.CustomParams.Set("category", "updated")
				cancel() // 触发重连
				return
			}
		}
	}()

	// 启动连接
	go client.Connect(ctx)

	// 保持主线程运行
	select {}
}
