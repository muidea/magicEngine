package test

import (
	"log"
	"net/http"

	engine "muidea.com/magicEngine"
)

// Hello hello middleware
type Hello struct {
}

// Handle handle request
func (s *Hello) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {

	ctx.Next()

	log.Print("Hello Handle")
}
