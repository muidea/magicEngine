package magicengine

import (
	"fmt"
	"log"
)

const serverName = "magic_engine"
const systemLogger = "systemLogger"
const systemStatic = "systemStatic"

func traceInfo(logger *log.Logger, info string) {
	logger.Printf("%s", info)
}

func panicInfo(info string) {
	msg := fmt.Sprintf("[%s] %s\n", serverName, info)
	panic(msg)
}
