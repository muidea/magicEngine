package http

import (
	"bufio"
	"errors"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
)

type hijackableResponseWriter struct {
	header nethttp.Header
	conn   net.Conn
	peer   net.Conn
}

func newHijackableResponseWriter() *hijackableResponseWriter {
	conn, peer := net.Pipe()
	return &hijackableResponseWriter{header: make(nethttp.Header), conn: conn, peer: peer}
}

func (w *hijackableResponseWriter) Header() nethttp.Header { return w.header }
func (w *hijackableResponseWriter) WriteHeader(int)        {}
func (w *hijackableResponseWriter) Write(body []byte) (int, error) {
	return len(body), nil
}
func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

// AetherRelay CP-WS-011: upgrades delegate hijacking and suppress post-101 writes.
func TestResponseWriterHijackDelegatesAndMarksWritten(t *testing.T) {
	underlying := newHijackableResponseWriter()
	defer underlying.conn.Close()
	defer underlying.peer.Close()

	wrapped := NewResponseWriter(underlying)
	hijacker, ok := wrapped.(nethttp.Hijacker)
	if !ok {
		t.Fatal("wrapped response writer does not implement http.Hijacker")
	}
	conn, buffer, err := hijacker.Hijack()
	if err != nil {
		t.Fatal(err)
	}
	if conn != underlying.conn || buffer == nil {
		t.Fatalf("hijack result conn=%v buffer=%v", conn, buffer)
	}
	if !wrapped.Written() || wrapped.Status() != nethttp.StatusSwitchingProtocols {
		t.Fatalf("written=%v status=%d", wrapped.Written(), wrapped.Status())
	}
}

func TestResponseWriterOptionalInterfacesAreSafe(t *testing.T) {
	underlying := httptest.NewRecorder()
	wrapped := NewResponseWriter(underlying)

	wrapped.(*responseWriter).Flush()
	if !wrapped.Written() || wrapped.Status() != nethttp.StatusOK || !underlying.Flushed {
		t.Fatalf("flush written=%v status=%d underlying_flushed=%v", wrapped.Written(), wrapped.Status(), underlying.Flushed)
	}
	if _, _, err := wrapped.(nethttp.Hijacker).Hijack(); !errors.Is(err, nethttp.ErrNotSupported) {
		t.Fatalf("Hijack error=%v, want http.ErrNotSupported", err)
	}
	if got := wrapped.(*responseWriter).Unwrap(); got != underlying {
		t.Fatalf("Unwrap()=%T, want original writer", got)
	}
}
