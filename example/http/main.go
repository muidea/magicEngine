package main

import (
	engine "github.com/muidea/magicEngine/http"
)

func main() {

	router := engine.NewRouteRegistry()

	Append(router)

	svr := engine.NewHTTPServer("8010", false)
	svr.Bind(router)

	svr.Use(&MiddleWareHello{Index: 100})
	svr.Use(&MiddleWareHello{Index: 101})
	svr.Use(&MiddleWareHello{Index: 102})

	//svr.Use(&test.Test{Index: 103})

	svr.Run()
}
