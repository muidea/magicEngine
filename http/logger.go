package http

import (
	"net/http"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
)

type logger struct {
	serialNo int64
}

func (s *logger) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	start := time.Now()

	addr := req.Header.Get("X-Real-IP")
	if addr == "" {
		addr = req.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = req.RemoteAddr
		}
	}

	s.serialNo++

	if EnableTrace() {
		log.Infof("Started-%05d %s %s for %s", s.serialNo, req.Method, req.URL.Path, addr)
	}

	rw := res.(ResponseWriter)
	ctx.Next()

	elapseVal := time.Since(start)
	if EnableTrace() {
		log.Infof("Completed-%05d %v %s in %v", s.serialNo, rw.Status(), http.StatusText(rw.Status()), elapseVal)
	} else if elapseVal >= GetElapseThreshold() {
		log.Warnf("Handle-%05d %s %s for %s %v in %v", s.serialNo, req.Method, req.URL.Path, addr, rw.Status(), elapseVal)
	}
}
