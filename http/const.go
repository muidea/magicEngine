package http

import (
	"fmt"

	"github.com/muidea/magicCommon/foundation/log"
)

const serverName = "magic_engine"
const systemStatic = "systemStatic"

func traceInfo(info string) {
	log.Infof(info)
}

func panicInfo(info string) {
	msg := fmt.Sprintf("[%s] %s\n", serverName, info)
	panic(msg)
}
