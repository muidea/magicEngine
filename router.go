package magicengine

import (
	"fmt"
	"log"
	"net/http"
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
	// Action 路由行为GET/PUT/POST/DELETE
	Method() string
	// Pattern 路由规则, 以'/'开始
	Pattern() string
	// Handler 路由处理器
	Handler() interface{}
}

// Router 路由器对象
type Router interface {
	// 增加路由
	AddRoute(rt Route, filters ...MiddleWareHandler)
	// 清除路由
	RemoveRoute(rt Route)
	// 分发一条请求
	Handle(ctx Context, res http.ResponseWriter, req *http.Request)
}

type rtItem struct {
	pattern string
	method  string
	handler interface{}
}

func (s *rtItem) Pattern() string {
	return s.pattern
}

func (s *rtItem) Method() string {
	return s.method
}

func (s *rtItem) Handler() interface{} {
	return s.handler
}

// CreateRoute create Route
func CreateRoute(pattern, method string, handler interface{}) Route {
	return &rtItem{pattern: pattern, method: method, handler: handler}
}

// 路由对象
type routeItem struct {
	route   Route
	filters []MiddleWareHandler
	regex   *regexp.Regexp
}

func (s *routeItem) equal(rt Route) bool {
	return s.route.Pattern() == rt.Pattern()
}

func (s *routeItem) match(path string) bool {
	matches := s.regex.FindStringSubmatch(path)
	if len(matches) > 0 && matches[0] == path {
		return true
	}

	return false
}

var routeReg1 = regexp.MustCompile(`:[^/#?()\.\\]+`)
var routeReg2 = regexp.MustCompile(`\*\*`)

func newRouteItem(rt Route, filters ...MiddleWareHandler) *routeItem {
	item := &routeItem{route: rt}
	item.filters = append(item.filters, filters...)

	pattern := routeReg1.ReplaceAllStringFunc(rt.Pattern(), func(m string) string {
		return fmt.Sprintf(`(?P<%s>[^/#?]+)`, m[1:])
	})
	var index int
	pattern = routeReg2.ReplaceAllStringFunc(pattern, func(m string) string {
		index++
		return fmt.Sprintf(`(?P<_%d>[^#?]*)`, index)
	})
	pattern += `\/?`
	item.regex = regexp.MustCompile(pattern)

	return item
}

type routeItemSlice []*routeItem

type router struct {
	routes     map[string]*routeItemSlice
	routesLock sync.RWMutex
}

// NewRouter 新建Router
func NewRouter() Router {
	return &router{routes: make(map[string]*routeItemSlice)}
}

func (s *router) AddRoute(rt Route, filters ...MiddleWareHandler) {
	ValidateRouteHandler(rt.Handler())
	for _, val := range filters {
		ValidateMiddleWareHandler(val)
	}

	log.Printf("[%s]:%s", rt.Method(), rt.Pattern())

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

		item := newRouteItem(rt, filters...)
		*routeSlice = append(*routeSlice, item)
		return
	}

	item := newRouteItem(rt, filters...)
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

func (s *router) Handle(ctx Context, res http.ResponseWriter, req *http.Request) {
	var routeSlice routeItemSlice
	func() {
		s.routesLock.RLock()
		defer s.routesLock.RUnlock()

		slice, ok := s.routes[strings.ToUpper(req.Method)]
		if ok {
			routeSlice = (*slice)[:]
		}
	}()

	var routeCtx RequestContext
	for _, val := range routeSlice {
		if val.match(req.URL.Path) {
			routeCtx = NewRouteContext(ctx, val.filters, val.route, res, req)
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
