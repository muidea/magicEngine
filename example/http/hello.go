package main

import (
	"context"
	"log/slog"
	"net/http"

	engine "github.com/muidea/magicEngine/http"
)

// helloKey is a type-safe key for context values
type helloKey struct{}

// MiddleWareHello hello middleware
type MiddleWareHello struct {
	Index int
}

// MiddleWareHandle handle request
func (s *MiddleWareHello) MiddleWareHandle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	curCtx := ctx.Context()

	slog.Info("middleware hello handle", "index", s.Index, "cur_value", curCtx.Value(helloKey{}))
	newCtx := context.WithValue(curCtx, helloKey{}, "1234")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	slog.Info("middleware hello handle completed", "index", s.Index)
}

// HelloMiddleWareRoute hello middleware
type HelloMiddleWareRoute struct {
	Index int
}

// MiddleWareHandle handle request
func (s *HelloMiddleWareRoute) MiddleWareHandle(ctx engine.RequestContext, res http.ResponseWriter, req *http.Request) {
	slog.Info("middleware hello route handle before", "index", s.Index)

	curCtx := ctx.Context()

	slog.Info("middleware hello route context", "index", s.Index, "cur_value", curCtx.Value(helloKey{}))
	newCtx := context.WithValue(curCtx, helloKey{}, "abcd")
	ctx.Update(newCtx)

	ctx.Next()

	ctx.Update(curCtx)
	slog.Info("middleware hello route handle after", "index", s.Index)
}
