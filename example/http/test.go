package main

import (
	"log/slog"
	"net/http"

	engine "github.com/muidea/magicEngine/http"
)

// Test hello middleware
type Test struct {
	Index int
}

// Handle handle request
func (s *Test) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	slog.Info("test handle", "index", s.Index)
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = res.Write([]byte("test world"))
}
