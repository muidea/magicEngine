package magicengine

import (
	"log"
	"net/http"
	"os"
)

// HTTPServer HTTPServer
type HTTPServer interface {
	Use(handler MiddleWareHandler)
	Bind(router Router)
	Run()
}

type httpServer struct {
	listenAddr string
	router     Router
	filter     MiddleWareChains
	logger     *log.Logger
}

// NewHTTPServer 新建HTTPServer
func NewHTTPServer(listenAddr string) HTTPServer {
	svr := &httpServer{listenAddr: listenAddr, filter: NewMiddleWareChains(), logger: log.New(os.Stdout, "[magic_engine] ", 0)}

	svr.Use(&logger{})
	svr.Use(&recovery{})

	return svr
}

func (s *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	ctx := NewRequestContext(s.filter.GetHandlers(), s.router, res, req)
	ctx.SetData(systemLogger, s.logger)

	ctx.Run()
}

func (s *httpServer) Use(handler MiddleWareHandler) {
	s.filter.Append(handler)
}

func (s *httpServer) Bind(router Router) {
	s.router = router
}

func (s *httpServer) Run() {
	traceInfo(s.logger, "listening on "+s.listenAddr)

	err := http.ListenAndServe(s.listenAddr, s)
	log.Fatalf("run httpserver fatal, err:%s", err.Error())
}
