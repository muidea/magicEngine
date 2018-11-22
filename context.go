package magicengine

import (
	"net/http"
	"reflect"
)

// Context request base context
type Context interface {
	SetData(key string, value interface{})
	GetData(key string) (interface{}, bool)
	RemoveData(key string)
}

// RequestContext represents a request context. Services can be mapped on the request level from this interface.
type RequestContext interface {
	Context
	// Next is an optional function that Middleware Handlers can call to yield the until after
	// the other Handlers have been executed. This works really well for any operations that must
	// happen after an http request
	Next()
	// Written returns whether or not the response for this context has been written.
	Written() bool
	// Run
	Run()
}

const (
	middleWareHandleParamNum = 4
	maxRouteHandleParamNum   = 3
	minRouteHandleParamNum   = 2
)

// ValidateMiddleWareHandler 校验MiddleWareHandler
func ValidateMiddleWareHandler(handler interface{}) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Ptr {
		panicInfo("middleware handler must be a callable interface")
	}

	handlerMethod, ok := handlerType.MethodByName("Handle")
	if !ok {
		panicInfo("middleware handler isn\\'t have Handle func")
	}

	methodType := handlerMethod.Type
	paramNum := methodType.NumIn()
	if paramNum != middleWareHandleParamNum {
		panicInfo("middleware handler invalid handle func param number")
	}

	// param0 := methodType.In(0).String()
	param1 := methodType.In(1)
	if param1.Kind() != reflect.Interface {
		panicInfo("middleware handler invalid handle func param0 type")
	}
	if param1.Name() != "RequestContext" {
		panicInfo("middleware handler invalid handle func param0 type")
	}
	param2 := methodType.In(2)
	if param2.Kind() != reflect.Interface {
		panicInfo("middleware handler invalid handle func param1 type")
	}
	if param2.String() != "http.ResponseWriter" {
		panicInfo("middleware handler invalid handle func param1 type")
	}

	param3 := methodType.In(3)
	if param3.Kind() != reflect.Ptr {
		panicInfo("middleware handler invalid handle func param2 type")
	}
	if param3.String() != "*http.Request" {
		panicInfo("middleware handler invalid handle func param2 type")
	}
}

// InvokeMiddleWareHandler 执行MiddleWareHandle
func InvokeMiddleWareHandler(handler interface{}, ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	params := make([]reflect.Value, middleWareHandleParamNum)
	params[0] = reflect.ValueOf(handler)
	params[1] = reflect.ValueOf(ctx)
	params[2] = reflect.ValueOf(res)
	params[3] = reflect.ValueOf(req)

	handlerType := reflect.TypeOf(handler)
	// 已经验证通过，所以这里就不用继续判断
	//if handlerType.Kind() != reflect.Ptr {
	//	panicInfo("middleware handler must be a callable interface")
	//}

	handlerMethod, _ := handlerType.MethodByName("Handle")
	// 已经验证通过，所以这里就不用继续判断
	//if !ok {
	//	panicInfo("middleware handler isn\\'t have Handle func")
	//}

	fv := handlerMethod.Func
	fv.Call(params)
}

// ValidateRouteHandler 校验RouteHandler
func ValidateRouteHandler(handler interface{}) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panicInfo("route handler must be a callable func")
	}

	paramNum := handlerType.NumIn()
	if paramNum == maxRouteHandleParamNum {
		param0 := handlerType.In(0)
		if param0.Kind() != reflect.Interface {
			panicInfo("route handler invalid handle func param0 type")
		}
		if param0.Name() != "Context" {
			panicInfo("route handler invalid handle func param0 type, expect: Context")
		}
		param1 := handlerType.In(1)
		if param1.Kind() != reflect.Interface {
			panicInfo("route handler invalid handle func param1 type")
		}
		if param1.String() != "http.ResponseWriter" {
			panicInfo("route handler invalid handle func param1 type, expect: http.ResponseWriter")
		}

		param2 := handlerType.In(2)
		if param2.Kind() != reflect.Ptr {
			panicInfo("route handler invalid handle func param2 type")
		}
		if param2.String() != "*http.Request" {
			panicInfo("route handler invalid handle func param2 type, expect: *http.Request")
		}
	} else if paramNum == minRouteHandleParamNum {
		param0 := handlerType.In(0)
		if param0.Kind() != reflect.Interface {
			panicInfo("route handler invalid handle func param0 type")
		}
		if param0.String() != "http.ResponseWriter" {
			panicInfo("route handler invalid handle func param0 type, expect: http.ResponseWriter")
		}

		param1 := handlerType.In(1)
		if param1.Kind() != reflect.Ptr {
			panicInfo("route handler invalid handle func param0 type")
		}
		if param1.String() != "*http.Request" {
			panicInfo("route handler invalid handle func param0 type, expect: *http.Request")
		}
	} else {
		panicInfo("illegal callable func")
	}
}

// InvokeRouteHandler 执行RouteHandle
func InvokeRouteHandler(handler interface{}, ctx Context, res http.ResponseWriter, req *http.Request) {
	handlerType := reflect.TypeOf(handler)
	// 已经验证通过，所以这里就不用继续判断
	//if handlerType.Kind() != reflect.Func {
	//	panicInfo("route handler must be a callable func")
	//}

	var params []reflect.Value
	paramNum := handlerType.NumIn()
	if paramNum == maxRouteHandleParamNum {
		params = make([]reflect.Value, maxRouteHandleParamNum)
		params[0] = reflect.ValueOf(ctx)
		params[1] = reflect.ValueOf(res)
		params[2] = reflect.ValueOf(req)
	} else if paramNum == minRouteHandleParamNum {
		params = make([]reflect.Value, minRouteHandleParamNum)
		params[0] = reflect.ValueOf(res)
		params[1] = reflect.ValueOf(req)
	} else {
		panicInfo("illegal callable func")
	}

	fv := reflect.ValueOf(handler)
	fv.Call(params)
}

type requestContext struct {
	filters []MiddleWareHandler
	rw      ResponseWriter
	req     *http.Request
	index   int

	router  Router
	dataMap map[string]interface{}
}

// NewRequestContext 新建Context
func NewRequestContext(filters []MiddleWareHandler, router Router, res http.ResponseWriter, req *http.Request) RequestContext {
	return &requestContext{filters: filters, router: router, rw: NewResponseWriter(res), req: req, index: 0, dataMap: make(map[string]interface{})}
}

func (c *requestContext) Next() {
	c.index++
	c.Run()
}

func (c *requestContext) Written() bool {
	return c.rw.Written()
}

func (c *requestContext) Run() {
	totalSizxe := len(c.filters)
	for c.index < totalSizxe {
		handler := c.filters[c.index]
		InvokeMiddleWareHandler(handler, c, c.rw, c.req)

		c.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() && c.router != nil {
		c.router.Handle(c, c.rw, c.req)
	} else {
		// 到这里说明没有router，也没有对应的MiddleWareHandler
		http.NotFound(c.rw, c.req)
		//http.Redirect(c.rw, c.req, "/404.html", http.StatusNotFound)
	}
}

func (c *requestContext) SetData(key string, value interface{}) {
	c.dataMap[key] = value
}

func (c *requestContext) GetData(key string) (interface{}, bool) {
	val, ok := c.dataMap[key]
	return val, ok
}

func (c *requestContext) RemoveData(key string) {
	delete(c.dataMap, key)
}

type routeContext struct {
	filters []MiddleWareHandler
	rw      ResponseWriter
	req     *http.Request
	index   int

	route   Route
	context Context
}

// NewRouteContext 新建Context
func NewRouteContext(reqCtx Context, filters []MiddleWareHandler, route Route, res http.ResponseWriter, req *http.Request) RequestContext {
	return &routeContext{filters: filters, route: route, rw: NewResponseWriter(res), req: req, index: 0, context: reqCtx}
}

func (c *routeContext) Next() {
	c.index++
	c.Run()
}

func (c *routeContext) Written() bool {
	return c.rw.Written()
}

func (c *routeContext) Run() {
	totalSizxe := len(c.filters)
	for c.index < totalSizxe {
		handler := c.filters[c.index]
		InvokeMiddleWareHandler(handler, c, c.rw, c.req)

		c.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() {
		InvokeRouteHandler(c.route.Handler(), c.context, c.rw, c.req)
	}
}

func (c *routeContext) SetData(key string, value interface{}) {
	c.context.SetData(key, value)
}

func (c *routeContext) GetData(key string) (interface{}, bool) {
	return c.context.GetData(key)
}

func (c *routeContext) RemoveData(key string) {
	c.context.RemoveData(key)
}
