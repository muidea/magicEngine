package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (s roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return s(req)
}

func TestCreateProxyRouteCachesProxyAndRewritesDynamicPath(t *testing.T) {
	route := CreateProxyRoute("/demo/:id", http.MethodGet, "https://backend.example/target/:id?fixed=1", true)
	proxyRoutePtr, ok := route.(*proxyRoute)
	if !ok {
		t.Fatalf("route type = %T, want *proxyRoute", route)
	}
	if proxyRoutePtr.proxy == nil {
		t.Fatal("expected proxy to be prebuilt at route creation")
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/demo/42?runtime=2", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set(DynamicTag, ":id")
	req.Header.Set(DynamicValue, "42")

	proxyRoutePtr.proxy.Director(req)

	if req.URL.Scheme != "https" {
		t.Fatalf("scheme = %q, want %q", req.URL.Scheme, "https")
	}
	if req.URL.Host != "backend.example" {
		t.Fatalf("host = %q, want %q", req.URL.Host, "backend.example")
	}
	if req.URL.Path != "/target/42" {
		t.Fatalf("path = %q, want %q", req.URL.Path, "/target/42")
	}
	gotQuery := req.URL.RawQuery
	if gotQuery != "fixed=1&runtime=2" && gotQuery != "runtime=2&fixed=1" {
		t.Fatalf("query = %q, want merged fixed/runtime params", gotQuery)
	}
}

func TestProxyHTTPForwardsRequestToDynamicTarget(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotQuery  string
		gotBody   string
	)
	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		gotMethod = req.Method
		gotPath = req.URL.Path
		gotQuery = req.URL.RawQuery
		gotBody = string(body)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("proxied")),
		}, nil
	})
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/gateway?runtime=2", strings.NewReader("payload"))
	req.Header.Set("Content-Type", "text/plain")
	res := httptest.NewRecorder()

	err := ProxyHTTP(res, req, "https://backend.example/target?fixed=1", nil)
	if err != nil {
		t.Fatalf("ProxyHTTP failed: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/target" {
		t.Fatalf("path = %q, want %q", gotPath, "/target")
	}
	if gotQuery != "fixed=1&runtime=2" && gotQuery != "runtime=2&fixed=1" {
		t.Fatalf("query = %q, want merged fixed/runtime params", gotQuery)
	}
	if gotBody != "payload" {
		t.Fatalf("body = %q, want %q", gotBody, "payload")
	}
	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusAccepted)
	}
	if res.Body.String() != "proxied" {
		t.Fatalf("response = %q, want %q", res.Body.String(), "proxied")
	}
}

func TestProxyHTTPRejectsIllegalTarget(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/gateway", nil)
	res := httptest.NewRecorder()

	err := ProxyHTTP(res, req, "://bad-target", nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
