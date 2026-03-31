package http

import (
	"net/http"
	"testing"
)

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
