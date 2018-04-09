package main

import (
	engine "muidea.com/magicEngine"
	"muidea.com/magicEngine/test"
)

func main() {

	router := engine.NewRouter()

	test.Append(router)

	svr := engine.NewHTTPServer(":8010")
	svr.Bind(router)

	svr.Use(&test.Hello{})

	svr.Use(&test.Test{})

	svr.Run()
}
