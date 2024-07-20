package main

import (
	"github.com/muidea/magicEngine/http"
	"github.com/muidea/magicEngine/test"
)

func main() {

	router := http.NewRouter()

	test.Append(router)

	svr := http.NewHTTPServer("8010")
	svr.Bind(router)

	svr.Use(&test.MiddleWareHello{Index: 100})
	svr.Use(&test.MiddleWareHello{Index: 101})
	svr.Use(&test.MiddleWareHello{Index: 102})

	//svr.Use(&test.Test{Index: 103})

	svr.Run()
}
