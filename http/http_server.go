package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/muidea/magicCommon/foundation/log"
)

// HTTPServer HTTPServer
type HTTPServer interface {
	Use(handler MiddleWareHandler)
	Bind(routeRegistry RouteRegistry)
	Run()
}

type httpServer struct {
	listenAddr    string
	routeRegistry RouteRegistry
	filter        MiddleWareChains
	staticOptions *StaticOptions
}

// NewHTTPServer 新建HTTPServer
func NewHTTPServer(bindPort string, enableStatic bool) HTTPServer {
	listenAddr := fmt.Sprintf(":%s", bindPort)
	svr := &httpServer{listenAddr: listenAddr, filter: NewMiddleWareChains()}

	svr.Use(&logger{})
	svr.Use(&recovery{})
	if enableStatic {
		svr.staticOptions = &StaticOptions{Path: "static", Prefix: "static"}
		svr.Use(&static{rootPath: Root})
	}

	return svr
}

func (s *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	valueContext := context.WithValue(req.Context(), systemStatic{}, s.staticOptions)
	ctx := NewRequestContext(s.filter.GetHandlers(), s.routeRegistry, valueContext, res, req)

	ctx.Run()
}

func (s *httpServer) Use(handler MiddleWareHandler) {
	s.filter.Append(handler)
}

func (s *httpServer) Bind(routeRegistry RouteRegistry) {
	s.routeRegistry = routeRegistry
}

func (s *httpServer) Run() {
	log.Infof("listening on %s", s.listenAddr)
	err := http.ListenAndServe(s.listenAddr, s)
	log.Criticalf("run httpserver fatal, err:%s", err.Error())
}
