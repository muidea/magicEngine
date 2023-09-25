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

	logPtr.Printf("Started %s %s for %s", req.Method, req.URL.Path, addr)

	rw := res.(ResponseWriter)
	ctx.Next()

	logPtr.Printf("Completed %v %s in %v\n", rw.Status(), http.StatusText(rw.Status()), time.Since(start))
}
