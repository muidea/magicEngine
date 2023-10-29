package test

import (
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// Test hello middleware
type Test struct {
	Index int
}

// Handle handle request
func (s *Test) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	log.Printf("Test Handle, index:%d", s.Index)
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("test world"))
}
