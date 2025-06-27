package http

import (
	"net/http"
	"sync"
)

// MiddleWareHandler 中间件处理器
type MiddleWareHandler interface {
	MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request)
}

// MiddleWareChains 处理器链
type MiddleWareChains interface {
	Append(handler MiddleWareHandler)

	GetHandlers() []MiddleWareHandler
}

type chainsImpl struct {
	handlers    []MiddleWareHandler
	handlesLock sync.RWMutex
}

// NewMiddleWareChains 新建MiddleWareChains
func NewMiddleWareChains() MiddleWareChains {
	return &chainsImpl{handlers: []MiddleWareHandler{}}
}

func (s *chainsImpl) GetHandlers() []MiddleWareHandler {
	s.handlesLock.RLock()
	defer s.handlesLock.RUnlock()

	return s.handlers[:]
}

func (s *chainsImpl) Append(handler MiddleWareHandler) {
	ValidateMiddleWareHandler(handler)

	s.handlesLock.Lock()
	defer s.handlesLock.Unlock()

	s.handlers = append(s.handlers, handler)
}
