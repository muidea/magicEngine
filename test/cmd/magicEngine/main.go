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

	svr.Use(&test.Hello{})

	//svr.Use(&test.Test{})

	svr.Run()
}
