package sse

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type sinkRecorder struct {
	closed bool
	events []Event
}

func (s *sinkRecorder) OnClose() {
	s.closed = true
}

func (s *sinkRecorder) OnRecv(event string, content []byte) {
	s.events = append(s.events, Event{Name: event, Data: append([]byte(nil), content...)})
}

func TestNewClientUsesRetryWait(t *testing.T) {
	client := NewClient("http://example.com", 2*time.Second, 3)
	if client.retryWait != 2*time.Second {
		t.Fatalf("expected retry wait to be set")
	}
}

func TestClientGetHonorsContextCancellation(t *testing.T) {
	client := NewClient("http://example.com/events", time.Millisecond, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Get(ctx, url.Values{}, &sinkRecorder{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestHolderRunNilTask(t *testing.T) {
	holder := NewHolder(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err := holder.Run(nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHolderRunPropagatesTaskError(t *testing.T) {
	holder := NewHolder(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	expected := errors.New("task failed")
	if err := holder.Run(func() error { return expected }); !errors.Is(err, expected) {
		t.Fatalf("expected task error, got %v", err)
	}
}

func TestClientRecvValParsesEventAndRetry(t *testing.T) {
	client := NewClient("http://example.com", time.Second, 1)
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("event: update\nid: 42\ndata: hello\nretry: 2s\n\n")),
	}
	sink := &sinkRecorder{}

	err := client.recvVal(context.Background(), resp, sink)
	if err != nil {
		t.Fatalf("recvVal failed: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	if sink.events[0].Name != "update" {
		t.Fatalf("unexpected event name: %q", sink.events[0].Name)
	}
	if !bytes.Contains(sink.events[0].Data, []byte("hello")) {
		t.Fatalf("unexpected event data: %q", string(sink.events[0].Data))
	}
	if client.retryWait != 2*time.Second {
		t.Fatalf("expected retry wait to update, got %v", client.retryWait)
	}
	if client.lastEventID != "42" {
		t.Fatalf("unexpected last event id: %q", client.lastEventID)
	}
}

func TestHolderOnRecvWritesCompleteSSEFrame(t *testing.T) {
	res := httptest.NewRecorder()
	holder := NewHolder(res, httptest.NewRequest(http.MethodGet, "/", nil))

	holder.OnRecv("update", []byte("payload"))

	body := res.Body.String()
	if !strings.Contains(body, "event: update\n") {
		t.Fatalf("unexpected event body: %q", body)
	}
	if !strings.Contains(body, "data: payload\n\n") {
		t.Fatalf("expected complete SSE frame, got %q", body)
	}
}

func TestHolderHeartbeatWritesCompleteFrame(t *testing.T) {
	res := httptest.NewRecorder()
	holder := NewHolder(res, httptest.NewRequest(http.MethodGet, "/", nil))

	if err := holder.heartbeat(); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	if res.Body.String() != ": ping\n\n" {
		t.Fatalf("unexpected heartbeat frame: %q", res.Body.String())
	}
}
