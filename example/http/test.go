package main

import (
	"net/http"

	"github.com/muidea/magicCommon/foundation/log"
	engine "github.com/muidea/magicEngine/http"
)

// Test hello middleware
type Test struct {
	Index int
}

// Handle handle request
func (s *Test) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	log.Infof("Test Handle, index:%d", s.Index)
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("test world"))
}
