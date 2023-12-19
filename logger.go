package magicengine

import (
	"log"
	"net/http"
	"time"
)

type logger struct {
}

func (s *logger) Handle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	obj := ctx.Context().Value(systemLogger)
	if obj == nil {
		panicInfo("cant\\'t get logger")
	}
	logPtr := obj.(*log.Logger)

	start := time.Now()

	addr := req.Header.Get("X-Real-IP")
	if addr == "" {
		addr = req.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = req.RemoteAddr
		}
	}

	if EnableTrace() {
		logPtr.Printf("Started %s %s for %s", req.Method, req.URL.Path, addr)
	}

	rw := res.(ResponseWriter)
	ctx.Next()

	elapseVal := time.Since(start)
	if EnableTrace() {
		logPtr.Printf("Completed %v %s in %v", rw.Status(), http.StatusText(rw.Status()), elapseVal)
	} else if elapseVal >= GetElapseThreshold() {
		logPtr.Printf("Handle %s %s for %s %v in %v", req.Method, req.URL.Path, addr, rw.Status(), elapseVal)
	}
}
