package test

import (
	"log"
	"net/http"

	engine "muidea.com/magicEngine"
)

// Append append router
func Append(router engine.Router) {
	router.AddRoute(&getRoute{})
}

type getRoute struct {
}

func (s *getRoute) Method() string {
	return "GET"
}

func (s *getRoute) Pattern() string {
	return "/demo/:id"
}

func (s *getRoute) Handler() interface{} {
	return s.getDemo
}

func (s *getRoute) getDemo(res http.ResponseWriter, req *http.Request) {
	log.Print("getDemo....")
}
