package main

import (
	engine "github.com/muidea/magicEngine"
	"github.com/muidea/magicEngine/test"
)

func main() {

	router := engine.NewRouter()

	test.Append(router)

	svr := engine.NewHTTPServer("8010")
	svr.Bind(router)

	svr.Use(&test.MiddleWareHello{Index: 100})
	svr.Use(&test.MiddleWareHello{Index: 101})
	svr.Use(&test.MiddleWareHello{Index: 102})

	svr.Use(&test.Test{})

	svr.Run()
}
