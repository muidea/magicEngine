package http

import (
	"fmt"
)

const serverName = "magic_engine"

type systemStatic struct{}

func panicInfo(info string) {
	msg := fmt.Sprintf("[%s] %s\n", serverName, info)
	panic(msg)
}
