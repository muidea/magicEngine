package http

import (
	"net/http"
	"testing"
)

func TestMiddlewareChains_Append(t *testing.T) {
	chains := NewMiddleWareChains()

	handler := &testMiddlewareHandler{}
	chains.Append(handler)

	handlers := chains.GetHandlers()
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}
}

func TestMiddlewareChains_MultipleHandlers(t *testing.T) {
	chains := NewMiddleWareChains()

	handler1 := &testMiddlewareHandler{}
	handler2 := &testMiddlewareHandler{}

	chains.Append(handler1)
	chains.Append(handler2)

	handlers := chains.GetHandlers()
	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}
}

func TestMiddlewareChains_ConcurrentAccess(t *testing.T) {
	chains := NewMiddleWareChains()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			handler := &testMiddlewareHandler{}
			chains.Append(handler)
			chains.GetHandlers()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

type testMiddlewareHandler struct{}

func (h *testMiddlewareHandler) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	ctx.Next()
}

func TestNewHTTPServer_WithOptions(t *testing.T) {
	server := NewHTTPServer(
		WithPort("9000"),
		WithStaticEnabled(true),
	)

	if server == nil {
		t.Error("expected non-nil server")
	}
}

func TestNewHTTPServer_DefaultPort(t *testing.T) {
	server := NewHTTPServer()

	if server == nil {
		t.Error("expected non-nil server")
	}
}
