package magicengine

import (
	"os"
)

// Envs
const (
	Dev  string = "development"
	Prod string = "production"
	Test string = "test"
)

// Env is the environment that Martini is executing in. The MAGICENGINE_ENV is read on initialization to set this variable.
var Env = Dev

// Root is current work path
var Root string

func setENV(e string) {
	if len(e) > 0 {
		Env = e
	}
}

func init() {
	setENV(os.Getenv("MAGICENGINE_ENV"))
	var err error
	Root, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}
