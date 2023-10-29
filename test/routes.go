package test

import (
	"context"
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// Append append router
func Append(router engine.Router) {
	router.AddRoute(&getRoute{}, &HelloMiddleWareRoute{Index: 200})

	router.AddRoute(&getRoute2{})

	router.AddRoute(engine.CreateProxyRoute("/proxy/abc", "GET", "http://127.0.0.1:8010/demo/12?ab=12", true))
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
	log.Print(req.URL)
	log.Print("getDemo....")
	log.Printf("hello=%v", ctx.Value("hello"))
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
	log.Print(req.URL)
	log.Print("getDemo2....")
	log.Printf("hello=%v", ctx.Value("hello"))
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("getDemo...."))
}
