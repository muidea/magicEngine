package http

import (
	"context"
	"net/http"
)

type RequestContext interface {
	Update(ctx context.Context)
	Context() context.Context
	Value(key any) any
	Next()
	Written() bool
	Run()
}

type requestContext struct {
	middlewareChainsFuncs []MiddleWareHandleFunc
	rw                     ResponseWriter
	req                    *http.Request
	index                  int

	routeRegistry RouteRegistry
	context       context.Context
}

// NewRequestContext 新建Context
func NewRequestContext(middlewareChains []MiddleWareHandleFunc, routeRegistry RouteRegistry, ctx context.Context, res http.ResponseWriter, req *http.Request) RequestContext {
	return &requestContext{
		middlewareChainsFuncs: middlewareChains,
		routeRegistry:          routeRegistry,
		context:                ctx,
		rw:                     NewResponseWriter(res),
		req:                    req,
		index:                  0,
	}
}

func (c *requestContext) Update(ctx context.Context) {
	c.context = ctx
}

func (c *requestContext) Context() context.Context {
	return c.context
}

func (c *requestContext) Value(key any) any {
	return c.context.Value(key)
}

func (c *requestContext) Next() {
	c.index++
	c.Run()
}

func (c *requestContext) Written() bool {
	return c.rw.Written()
}

func (c *requestContext) Run() {
	totalSize := len(c.middlewareChainsFuncs)
	// 先执行中间件
	for c.index < totalSize {
		c.middlewareChainsFuncs[c.index](c, c.rw, c.req)
		//InvokeMiddleWareHandler(handler, c, c.rw, c.req)

		c.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() && c.routeRegistry != nil {
		c.routeRegistry.Handle(c.context, c.rw.(http.ResponseWriter), c.req)
		if !c.Written() {
			http.Error(c.rw, "", http.StatusNoContent)
		}
	} else {
		// 到这里说明没有router，也没有对应的MiddleWareHandler
		http.NotFound(c.rw, c.req)
		//http.Redirect(c.rw, c.req, "/404.html", http.StatusNotFound)
	}
}

type routeContext struct {
	middlewareChainsHandler []MiddleWareHandler
	rw                       ResponseWriter
	req                      *http.Request
	index                    int

	route   Route
	context context.Context
}

// NewRouteContext 新建Context
func NewRouteContext(reqCtx context.Context, chainsHandler []MiddleWareHandler, route Route, res http.ResponseWriter, req *http.Request) RequestContext {
	return &routeContext{
		middlewareChainsHandler: chainsHandler,
		route:                    route,
		rw:                       res.(ResponseWriter),
		req:                      req,
		index:                    0,
		context:                  reqCtx,
	}
}

func (c *routeContext) Update(ctx context.Context) {
	c.context = ctx
}

func (c *routeContext) Context() context.Context {
	return c.context
}

func (c *routeContext) Value(key any) any {
	return c.context.Value(key)
}

func (c *routeContext) Next() {
	c.index++
	c.Run()
}

func (c *routeContext) Written() bool {
	return c.rw.Written()
}

func (c *routeContext) Run() {
	totalSize := len(c.middlewareChainsHandler)
	for c.index < totalSize {
		c.middlewareChainsHandler[c.index].MiddleWareHandle(c, c.rw, c.req)
		c.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() {
		funHandle := c.route.Handler()
		funHandle(c.context, c.rw, c.req)
		//InvokeRouteHandler(c.route.Handler(), c.context, c.rw, c.req)
	}

	if !c.Written() {
		http.Error(c.rw, "", http.StatusNoContent)
	}
}
