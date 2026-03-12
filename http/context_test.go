package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestContext_Basic(t *testing.T) {
	registry := NewRouteRegistry()
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	registry.AddHandler("/test", GET, handler)

	_ = httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chains := NewMiddleWareChains()
	ctx := context.Background()
	_ = NewRequestContext(chains.GetHandlers(), registry, ctx, w, nil)
}

func TestResponseWriter_Status(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	rw := NewResponseWriter(w)

	if rw.Status() != 0 {
		t.Errorf("expected initial status 0, got %d", rw.Status())
	}

	rw.WriteHeader(http.StatusOK)

	if rw.Status() != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rw.Status())
	}
}

func TestResponseWriter_Written(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	rw := NewResponseWriter(w)

	if rw.Written() {
		t.Error("expected Written() to be false initially")
	}

	rw.WriteHeader(http.StatusOK)

	if !rw.Written() {
		t.Error("expected Written() to be true after WriteHeader")
	}
}

func TestResponseWriter_Size(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	rw := NewResponseWriter(w)

	body := []byte("test body")
	rw.Write(body)

	if rw.Size() != len(body) {
		t.Errorf("expected size %d, got %d", len(body), rw.Size())
	}
}

func TestResponseWriter_Write(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	rw := NewResponseWriter(w)

	body := []byte("test body")
	n, err := rw.Write(body)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(body) {
		t.Errorf("expected %d bytes written, got %d", len(body), n)
	}
	if rw.Status() != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rw.Status())
	}
}
