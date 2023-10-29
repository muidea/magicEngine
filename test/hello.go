package test

import (
	"context"
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// MiddleWareHello hello middleware
type MiddleWareHello struct {
	Index int
}

// Handle handle request
func (s *MiddleWareHello) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	curCtx := ctx.Context()

	log.Printf("MiddleWareHello Handle, index:%d,curValue:%v", s.Index, curCtx.Value("hello"))
	newCtx := context.WithValue(curCtx, "hello", "1234")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	log.Printf("MiddleWareHello Handle, index:%d", s.Index)
}

// HelloMiddleWareRoute hello middleware
type HelloMiddleWareRoute struct {
	Index int
}

// Handle handle request
func (s *HelloMiddleWareRoute) Handle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	log.Printf("MiddleWareHello Route Handle before, index:%d", s.Index)

	curCtx := ctx.Context()

	log.Printf("index:%d,curValue:%v", s.Index, curCtx.Value("hello"))
	newCtx := context.WithValue(curCtx, "hello", "abcd")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	log.Printf("MiddleWareHello Route Handle after, index:%d", s.Index)
}
