package test

import (
	"log"
	"net/http"

	engine "github.com/muidea/magicEngine"
)

// Append append router
func Append(router engine.Router) {
	router.AddRoute(&getRoute{}, &HelloRoute{})

	router.AddRoute(engine.CreateProxyRoute("/proxy/abc", "GET", "http://127.0.0.1:8010/demo/12?ab=12"))
}

type getRoute struct {
}

func (s *getRoute) Method() string {
	return "GET"
}

func (s *getRoute) Pattern() string {
	return "/demo/:id"
}

func (s *getRoute) Handler() func(http.ResponseWriter, *http.Request) {
	return s.getDemo
}

func (s *getRoute) getDemo(res http.ResponseWriter, req *http.Request) {
	log.Print(req.URL)
	log.Print("getDemo....")
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Write([]byte("getDemo...."))
}
