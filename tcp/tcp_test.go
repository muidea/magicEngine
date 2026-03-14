package tcp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/muidea/magicCommon/execute"
)

type observerRecorder struct {
	mu          sync.Mutex
	connects    int
	disconnects int
	payloads    [][]byte
}

func (s *observerRecorder) OnConnect(ep Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connects++
}

func (s *observerRecorder) OnDisConnect(ep Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnects++
}

func (s *observerRecorder) OnRecvData(ep Endpoint, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloads = append(s.payloads, append([]byte(nil), data...))
}

type partialWriteConn struct {
	writes [][]byte
	closed bool
}

func (s *partialWriteConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (s *partialWriteConn) Close() error                       { s.closed = true; return nil }
func (s *partialWriteConn) LocalAddr() net.Addr                { return stubAddr("local") }
func (s *partialWriteConn) RemoteAddr() net.Addr               { return stubAddr("remote") }
func (s *partialWriteConn) SetDeadline(t time.Time) error      { return nil }
func (s *partialWriteConn) SetReadDeadline(t time.Time) error  { return nil }
func (s *partialWriteConn) SetWriteDeadline(t time.Time) error { return nil }
func (s *partialWriteConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	chunk := 2
	if len(b) < chunk {
		chunk = len(b)
	}
	s.writes = append(s.writes, append([]byte(nil), b[:chunk]...))
	return chunk, nil
}

type errorWriteConn struct{}

func (s *errorWriteConn) Read(b []byte) (int, error)         { return 0, io.EOF }
func (s *errorWriteConn) Close() error                       { return nil }
func (s *errorWriteConn) LocalAddr() net.Addr                { return stubAddr("local") }
func (s *errorWriteConn) RemoteAddr() net.Addr               { return stubAddr("remote") }
func (s *errorWriteConn) SetDeadline(t time.Time) error      { return nil }
func (s *errorWriteConn) SetReadDeadline(t time.Time) error  { return nil }
func (s *errorWriteConn) SetWriteDeadline(t time.Time) error { return nil }
func (s *errorWriteConn) Write(b []byte) (int, error)        { return 0, errors.New("write failed") }

type stubAddr string

func (s stubAddr) Network() string { return "tcp" }
func (s stubAddr) String() string  { return string(s) }

func TestEndpointSendDataHandlesPartialWrites(t *testing.T) {
	conn := &partialWriteConn{}
	ep := newEndpoint(conn, nil)

	if err := ep.SendData([]byte("abcdef")); err != nil {
		t.Fatalf("send data failed: %v", err)
	}

	got := bytes.Join(conn.writes, nil)
	if string(got) != "abcdef" {
		t.Fatalf("unexpected written data: %q", string(got))
	}
}

func TestEndpointSendDataNotifiesDisconnectOnWriteError(t *testing.T) {
	observer := &observerRecorder{}
	ep := newEndpoint(&errorWriteConn{}, observer)

	err := ep.SendData([]byte("abc"))
	if err == nil {
		t.Fatal("expected write error")
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.disconnects != 1 {
		t.Fatalf("expected 1 disconnect, got %d", observer.disconnects)
	}
}

func TestEndpointRecvDataNotifiesObserver(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	observer := &observerRecorder{}
	ep := newEndpoint(serverConn, observer)

	done := make(chan error, 1)
	go func() {
		done <- ep.RecvData()
	}()

	if _, err := clientConn.Write([]byte("ping")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	_ = clientConn.Close()

	if err := <-done; err != nil {
		t.Fatalf("recv data failed: %v", err)
	}

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.connects != 1 {
		t.Fatalf("expected 1 connect, got %d", observer.connects)
	}
	if observer.disconnects != 1 {
		t.Fatalf("expected 1 disconnect, got %d", observer.disconnects)
	}
	if len(observer.payloads) != 1 || string(observer.payloads[0]) != "ping" {
		t.Fatalf("unexpected payloads: %#v", observer.payloads)
	}
}

func TestClientSendDataBeforeConnectFails(t *testing.T) {
	client := NewClient(nil)
	if err := client.SendData([]byte("abc")); err == nil {
		t.Fatal("expected error before connect")
	}
}

func TestSimpleEndpointManagerTracksLifecycle(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	observer := &observerRecorder{}
	execVal := execute.NewExecute(10)
	manager := NewEndpointManger(observer, &execVal)

	done := make(chan struct{})
	go func() {
		manager.OnNewConnect(serverConn)
		close(done)
	}()

	if _, err := clientConn.Write([]byte("hello")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	_ = clientConn.Close()
	<-done

	_ = manager.executePtr.WaitTimeout(time.Second)

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.connects != 1 {
		t.Fatalf("expected 1 connect, got %d", observer.connects)
	}
	if observer.disconnects != 1 {
		t.Fatalf("expected 1 disconnect, got %d", observer.disconnects)
	}
	if len(observer.payloads) != 1 || string(observer.payloads[0]) != "hello" {
		t.Fatalf("unexpected payloads: %#v", observer.payloads)
	}
}
