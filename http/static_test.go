package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRouteRegistry_RemoveLastRoute(t *testing.T) {
	registry := NewRouteRegistry()

	first := CreateRoute("/first", GET, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})
	second := CreateRoute("/second", GET, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})

	registry.AddRoute(first)
	registry.AddRoute(second)
	registry.RemoveRoute(second)

	if !registry.ExistRoute(first) {
		t.Fatal("expected first route to remain")
	}
	if registry.ExistRoute(second) {
		t.Fatal("expected second route to be removed")
	}
}

func TestServeStaticFileDirectoryFallback(t *testing.T) {
	rootDir := t.TempDir()
	dirPath := filepath.Join(rootDir, "docs")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	indexPath := filepath.Join(dirPath, "index.html")
	if err := os.WriteFile(indexPath, []byte("hello static"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	res := httptest.NewRecorder()

	err := serveStaticFile(http.Dir(rootDir), prepareStaticOptions(&StaticOptions{}), "/docs/", res, req, true)
	if err != nil {
		t.Fatalf("serve static file failed: %v", err)
	}

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if body := res.Body.String(); body != "hello static" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestServeStaticFileFallbackFile(t *testing.T) {
	rootDir := t.TempDir()
	fallbackPath := filepath.Join(rootDir, "index.html")
	if err := os.WriteFile(fallbackPath, []byte("fallback"), 0o644); err != nil {
		t.Fatalf("write fallback failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()
	opt := prepareStaticOptions(&StaticOptions{Fallback: "/index.html"})

	err := serveStaticFile(http.Dir(rootDir), opt, "/missing", res, req, true)
	if err != nil {
		t.Fatalf("serve static file failed: %v", err)
	}

	body, readErr := io.ReadAll(res.Result().Body)
	if readErr != nil {
		t.Fatalf("read body failed: %v", readErr)
	}
	if string(body) != "fallback" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestStaticHandlerPreservesMissingAndIOErrorSemantics(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "indexdir", "index.html"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "proof.bin"), []byte("original bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loop", filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/static/missing.bin", 404}, {"/static/missing.txt", 404}, {"/static/missing.md", 404}, {"/static/missing", 404},
		{"/static/empty/", 404}, {"/static/indexdir/", 404}, {"/static/loop", 500}, {"/static/proof.bin", 200},
		{"/static-other/proof.bin", 404},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), StaticOptionsKey{}, &StaticOptions{RootPath: root, PrefixUri: "/static"})
			response := httptest.NewRecorder()
			StaticHandler(ctx, response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status=%d expected=%d", response.Code, tc.status)
			}
			if tc.status == 200 && response.Body.String() != "original bytes" {
				t.Fatal("content changed")
			}
		})
	}
	ctx := context.WithValue(context.Background(), StaticOptionsKey{}, &StaticOptions{RootPath: root, PrefixUri: "/static"})
	request := httptest.NewRequest(http.MethodGet, "/static/proof.bin", nil)
	request.Header.Set("Range", "bytes=0-7")
	response := httptest.NewRecorder()
	StaticHandler(ctx, response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "original" {
		t.Fatal("range read changed")
	}
	if err := os.Remove(filepath.Join(root, "proof.bin")); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	StaticHandler(ctx, response, httptest.NewRequest(http.MethodGet, "/static/proof.bin", nil))
	if response.Code != http.StatusNotFound {
		t.Fatal("deleted bytes did not return 404")
	}
}
