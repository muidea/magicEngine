package main

import (
	"github.com/muidea/magicEngine/http"
	http2 "github.com/muidea/magicEngine/test/http"
)

func main() {

	router := http.NewRouteRegistry()

	http2.Append(router)

	svr := http.NewHTTPServer("8010", false)
	svr.Bind(router)

	svr.Use(&http2.MiddleWareHello{Index: 100})
	svr.Use(&http2.MiddleWareHello{Index: 101})
	svr.Use(&http2.MiddleWareHello{Index: 102})

	//svr.Use(&test.Test{Index: 103})

	svr.Run()
}
