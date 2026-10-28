package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestRoutesMatchDecodedLiteralFilenameCharacters(t *testing.T) {
	for _, pattern := range []string{"/static/private/**", "/static/private/:name"} {
		registry := NewRouteRegistry()
		registry.AddHandler(pattern, GET, func(ctx context.Context, w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Matched-Path", req.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		})
		for _, name := range []string{"report%23one.txt", "report%3Fone.txt", "report%25one.txt", "report%252Fone.txt"} {
			req := httptest.NewRequest(GET, "/static/private/"+name+"?fileToken=opaque", nil)
			res := httptest.NewRecorder()
			registry.Handle(context.Background(), NewResponseWriter(res), req)
			if res.Code != http.StatusNoContent || res.Header().Get("Matched-Path") != req.URL.Path {
				t.Fatalf("pattern %s file %s: status %d path %q", pattern, name, res.Code, res.Header().Get("Matched-Path"))
			}
		}
	}
}

func TestStaticWildcardServesLiteralEncodedFilenames(t *testing.T) {
	root := t.TempDir()
	registry := NewRouteRegistry()
	registry.AddHandler("/static/private/**", GET, func(ctx context.Context, w http.ResponseWriter, req *http.Request) {
		ctx = context.WithValue(ctx, StaticOptionsKey{}, &StaticOptions{RootPath: root, PrefixUri: "/static/private", SkipLogging: true})
		StaticHandler(ctx, w, req)
	})
	for _, name := range []string{"report #1.txt", "report ?1.txt", "report %1.txt", "report %2F.txt"} {
		content := []byte("exact bytes: " + name)
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(GET, "/static/private/"+url.PathEscape(name)+"?fileToken=opaque", nil)
		res := httptest.NewRecorder()
		registry.Handle(context.Background(), NewResponseWriter(res), req)
		if res.Code != http.StatusOK || !bytes.Equal(res.Body.Bytes(), content) {
			t.Fatal(name, res.Code, res.Body.String())
		}
	}
}
