package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// HTTPServer HTTPServer
type HTTPServer interface {
	Use(handler MiddleWareHandler)
	Bind(routeRegistry RouteRegistry)
	Run()
}

type httpServer struct {
	listenAddr       string
	routeRegistry    RouteRegistry
	middlewareChains MiddleWareChains
	staticOptions    *StaticOptions
}

// NewHTTPServer 新建HTTPServer
func NewHTTPServer(bindPort string, enableShareStatic bool) HTTPServer {
	listenAddr := fmt.Sprintf(":%s", bindPort)
	svr := &httpServer{listenAddr: listenAddr, middlewareChains: NewMiddleWareChains()}

	svr.Use(&logger{})
	svr.Use(&recovery{})

	svr.staticOptions = &StaticOptions{RootPath: "./static", PrefixUri: "/static", ExcludeUri: "/api/"}
	if enableShareStatic {
		svr.Use(&static{rootPath: Root})
	}

	return svr
}

func (s *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	httpContext := context.WithValue(req.Context(), StaticOptionsKey{}, s.staticOptions)
	ctx := NewRequestContext(s.middlewareChains.GetHandlers(), s.routeRegistry, httpContext, res, req)

	ctx.Run()
}

func (s *httpServer) Use(handler MiddleWareHandler) {
	s.middlewareChains.Append(handler)
}

func (s *httpServer) Bind(routeRegistry RouteRegistry) {
	s.routeRegistry = routeRegistry
}

func (s *httpServer) Run() {
	slog.Info("server listening", "addr", s.listenAddr)
	err := http.ListenAndServe(s.listenAddr, s)
	slog.Error("server fatal error", "err", err)
}
