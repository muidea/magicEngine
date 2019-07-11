package test

import (
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// Hello hello middleware
type Hello struct {
}

// Handle handle request
func (s *Hello) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {

	ctx.Next()

	log.Print("Hello Handle")
}

// HelloRoute hello middleware
type HelloRoute struct {
}

// Handle handle request
func (s *HelloRoute) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	log.Print("Hello Route Handle before")

	ctx.Next()

	log.Print("Hello Route Handle after")
}
