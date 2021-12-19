package test

import (
	"context"
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// MiddleWareHello hello middleware
type MiddleWareHello struct {
}

// Handle handle request
func (s *MiddleWareHello) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	curCtx := ctx.Context()

	log.Printf("curValue:%v", curCtx.Value("hello"))
	newCtx := context.WithValue(curCtx, "hello", "1234")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	log.Print("MiddleWareHello Handle")
}

// HelloMiddleWareRoute hello middleware
type HelloMiddleWareRoute struct {
}

// Handle handle request
func (s *HelloMiddleWareRoute) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	log.Print("MiddleWareHello Route Handle before")

	curCtx := ctx.Context()

	log.Printf("curValue:%v", curCtx.Value("hello"))
	newCtx := context.WithValue(curCtx, "hello", "abcd")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	log.Print("MiddleWareHello Route Handle after")
}
