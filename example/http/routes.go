package main

import (
	"context"
	"net/http"

	"github.com/muidea/magicCommon/foundation/log"
	engine "github.com/muidea/magicEngine/http"
)

// Append append routeRegistry
func Append(routeRegistry engine.RouteRegistry) {
	routeRegistry.AddRoute(&getRoute{}, &HelloMiddleWareRoute{Index: 200})

	routeRegistry.AddRoute(&getRoute2{})

	routeRegistry.AddRoute(engine.CreateProxyRoute("/proxy/abc", "GET", "http://127.0.0.1:8010/demo/12?ab=12", true))
}

type getRoute struct {
}

func (s *getRoute) Method() string {
	return "GET"
}

func (s *getRoute) Pattern() string {
	return "/demo/:id"
}

func (s *getRoute) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return s.getDemo
}

func (s *getRoute) getDemo(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	log.Infof(req.URL.String())
	log.Infof("getDemo....")
	log.Infof("hello=%v", ctx.Value("hello"))
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("getDemo...."))
}

type getRoute2 struct {
}

func (s *getRoute2) Method() string {
	return "GET"
}

func (s *getRoute2) Pattern() string {
	return "/hello/:id"
}

func (s *getRoute2) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return s.getDemo
}

func (s *getRoute2) getDemo(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	log.Infof(req.URL.String())
	log.Infof("getDemo2....")
	log.Infof("hello=%v", ctx.Value("hello"))
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("getDemo...."))
}
