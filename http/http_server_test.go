package http

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestHTTPServerRunReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().String()
	port = port[strings.LastIndex(port, ":")+1:]
	server := NewHTTPServer(WithPort(port))
	if err := server.Run(); err == nil {
		t.Fatal("expected listen error")
	}
}

func TestHTTPServerShutdownStopsRun(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port failed: %v", err)
	}
	addr := listener.Addr().String()
	port := addr[strings.LastIndex(addr, ":")+1:]
	if err := listener.Close(); err != nil {
		t.Fatalf("release port failed: %v", err)
	}

	server := NewHTTPServer(WithPort(port)).(*httpServer)
	done := make(chan error, 1)
	go func() {
		done <- server.Run()
	}()

	deadline := time.Now().Add(time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 10*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("run returned error after shutdown: %v", err)
	}
}
