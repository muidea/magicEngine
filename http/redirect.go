package http

import (
	"context"
	"net/http"
)

type redirectRoute struct {
	pattern         string
	method          string
	redirectPattern string
}

func (s *redirectRoute) Method() string {
	return s.method
}

func (s *redirectRoute) Pattern() string {
	return s.pattern
}

func (s *redirectRoute) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, s.redirectPattern, http.StatusSeeOther)
	}
}

func CreateRedirectRoute(pattern, method, redirectPattern string) Route {
	return &redirectRoute{pattern: pattern, method: method, redirectPattern: redirectPattern}
}
