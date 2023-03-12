package magicengine

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// 基本HTTP行为定义
const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	OPTIONS = "OPTIONS"
)

// Route 路由接口
type Route interface {
	// Method 路由行为GET/PUT/POST/DELETE
	Method() string
	// Pattern 路由规则, 以'/'开始
	Pattern() string
	// Handler 路由处理器
	Handler() func(context.Context, http.ResponseWriter, *http.Request)
}

// Router 路由器对象
type Router interface {
	// SetApiVersion 设置ApiVersion
	SetApiVersion(version string)
	// GetApiVersion 查询ApiVersion
	GetApiVersion() string
	// AddRoute 增加路由
	AddRoute(rt Route, filters ...MiddleWareHandler)
	// RemoveRoute 清除路由
	RemoveRoute(rt Route)
	// Handle 分发一条请求
	Handle(ctx context.Context, res http.ResponseWriter, req *http.Request)
}

type rtItem struct {
	pattern string
	method  string
	handler func(context.Context, http.ResponseWriter, *http.Request)
}

func (s *rtItem) Pattern() string {
	return s.pattern
}

func (s *rtItem) Method() string {
	return s.method
}

func (s *rtItem) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return s.handler
}

// CreateRoute create Route
func CreateRoute(pattern, method string, handler func(context.Context, http.ResponseWriter, *http.Request)) Route {
	return &rtItem{pattern: pattern, method: method, handler: handler}
}

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

type proxyRoute struct {
	pattern    string
	method     string
	reallyURL  string
	rewriteURL bool
}

func (s *proxyRoute) Pattern() string {
	return s.pattern
}

func (s *proxyRoute) Method() string {
	return s.method
}

func (s *proxyRoute) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return s.proxyFun
}

func (s *proxyRoute) proxyFun(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	url, err := url.Parse(s.reallyURL)
	if err != nil {
		log.Fatalf("illegal proxy really url, url:%s", s.reallyURL)
		return
	}

	if url.Hostname() == "" {
		if url.RawQuery != "" {
			url.RawQuery = url.RawQuery + "&" + req.URL.RawQuery
		} else {
			url.RawQuery = req.URL.RawQuery
		}

		http.Redirect(res, req, url.String(), http.StatusSeeOther)
		return
	}

	errorHandler := func(res http.ResponseWriter, req *http.Request, err error) {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
	}

	if s.rewriteURL {
		proxy := newReverseProxy(url)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	} else {
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	}
}

// CreateProxyRoute create proxy route
func CreateProxyRoute(pattern, method, reallyURL string, rewriteURL bool) Route {
	return &proxyRoute{pattern: pattern, method: method, reallyURL: reallyURL, rewriteURL: rewriteURL}
}

// PatternFilter route filter
type PatternFilter struct {
	regex *regexp.Regexp
}

var routeReg1 = regexp.MustCompile(`:[^/#?()\.\\]+`)
var routeReg2 = regexp.MustCompile(`\*\*`)

// NewPatternFilter new route filter
func NewPatternFilter(routePattern string) *PatternFilter {
	filter := &PatternFilter{}
	pattern := routeReg1.ReplaceAllStringFunc(routePattern, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[^/#?]+)`, m[1:])
	})
	var index int
	pattern = routeReg2.ReplaceAllStringFunc(pattern, func(m string) string {
		index++
		return fmt.Sprintf(`(?P<_%d>[^#?]*)`, index)
	})
	pattern += `\/?`
	filter.regex = regexp.MustCompile(pattern)

	return filter
}

// Match match path
func (s *PatternFilter) Match(path string) bool {
	matches := s.regex.FindStringSubmatch(path)
	if len(matches) > 0 && matches[0] == path {
		return true
	}

	return false
}

// 路由对象
type routeItem struct {
	route          Route
	middlewareList []MiddleWareHandler
	patternFilter  *PatternFilter
}

func (s *routeItem) equal(rt Route) bool {
	return s.route.Pattern() == rt.Pattern()
}

func (s *routeItem) match(path string) bool {
	return s.patternFilter.Match(path)
}

type routeItemSlice []*routeItem

type router struct {
	currentApiVersion string
	routes            map[string]*routeItemSlice
	routesLock        sync.RWMutex
}

// NewRouter 新建Router
func NewRouter() Router {
	return &router{routes: make(map[string]*routeItemSlice)}
}

func (s *router) SetApiVersion(version string) {
	s.currentApiVersion = version
}

func (s *router) GetApiVersion() string {
	return s.currentApiVersion
}

func (s *router) newRouteItem(rt Route, filters ...MiddleWareHandler) *routeItem {
	item := &routeItem{route: rt}
	item.middlewareList = append(item.middlewareList, filters...)
	rtPattern := rt.Pattern()
	if s.currentApiVersion != "" {
		rtPattern = fmt.Sprintf("%s%s", s.currentApiVersion, rtPattern)
	}
	item.patternFilter = NewPatternFilter(rtPattern)

	log.Printf("[%s]:%s", rt.Method(), rtPattern)

	return item
}

func (s *router) AddRoute(rt Route, filters ...MiddleWareHandler) {
	ValidateRouteHandler(rt.Handler())
	for _, val := range filters {
		ValidateMiddleWareHandler(val)
	}

	s.routesLock.Lock()
	defer s.routesLock.Unlock()

	routeSlice, ok := s.routes[rt.Method()]
	if ok {
		for _, val := range *routeSlice {
			if val.equal(rt) {
				msg := fmt.Sprintf("duplicate route!, pattern:%s, method:%s", rt.Pattern(), rt.Method())
				panicInfo(msg)
			}
		}

		item := s.newRouteItem(rt, filters...)
		*routeSlice = append(*routeSlice, item)
		return
	}

	item := s.newRouteItem(rt, filters...)
	routeSlice = &routeItemSlice{}
	*routeSlice = append(*routeSlice, item)
	s.routes[rt.Method()] = routeSlice
}

func (s *router) RemoveRoute(rt Route) {
	s.routesLock.Lock()
	defer s.routesLock.Unlock()

	routeSlice, ok := s.routes[rt.Method()]
	if !ok {
		msg := fmt.Sprintf("no found route!, pattern:%s, method:%s", rt.Pattern(), rt.Method())
		panicInfo(msg)
	}

	newRoutes := routeItemSlice{}
	for idx, val := range *routeSlice {
		if val.equal(rt) {
			if idx > 0 {
				newRoutes = append(newRoutes, (*routeSlice)[0:idx]...)
			}

			idx++
			if idx < len(s.routes) {
				newRoutes = append(newRoutes, (*routeSlice)[idx:]...)
			}

			break
		}
	}

	s.routes[rt.Method()] = &newRoutes
}

func (s *router) Handle(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	var routeSlice routeItemSlice
	func() {
		s.routesLock.RLock()
		defer s.routesLock.RUnlock()

		slice, ok := s.routes[strings.ToUpper(req.Method)]
		if ok {
			routeSlice = (*slice)[:]
		}
	}()

	// set default content-type = "application/json; charset=utf-8"
	//res.Header().Set("Content-Type", "application/json; charset=utf-8")
	var routeCtx RequestContext
	for _, val := range routeSlice {
		if val.match(req.URL.Path) {
			routeCtx = NewRouteContext(ctx, val.middlewareList, val.route, res, req)
			break
		}
	}

	if routeCtx != nil {
		routeCtx.Run()
		return
	}

	http.NotFound(res, req)
	//http.Redirect(res, req, "/404.html", http.StatusMovedPermanently)
}
