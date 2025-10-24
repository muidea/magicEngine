package http

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/muidea/magicCommon/foundation/log"
)

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
	uriPattern    string
	method     string
	reallyURL  string
	rewriteURL bool
}

func (s *proxyRoute) Pattern() string {
	return s.uriPattern
}

func (s *proxyRoute) Method() string {
	return s.method
}

func (s *proxyRoute) Handler() func(context.Context, http.ResponseWriter, *http.Request) {
	return s.proxyFun
}

func (s *proxyRoute) proxyFun(_ context.Context, res http.ResponseWriter, req *http.Request) {
	urlVal, err := url.Parse(s.reallyURL)
	if err != nil {
		log.Criticalf("illegal proxy really url, url:%s", s.reallyURL)
		return
	}

	dynamicTAG := req.Header.Get(DynamicTag)
	dynamicValue := req.Header.Get(DynamicValue)
	if dynamicTAG != "" && dynamicValue != "" {
		urlVal.Path = strings.Replace(urlVal.Path, dynamicTAG, dynamicValue, -1)
	}

	if urlVal.Hostname() == "" {
		if urlVal.RawQuery != "" {
			urlVal.RawQuery = urlVal.RawQuery + "&" + req.URL.RawQuery
		} else {
			urlVal.RawQuery = req.URL.RawQuery
		}

		http.Redirect(res, req, urlVal.String(), http.StatusSeeOther)
		return
	}

	errorHandler := func(res http.ResponseWriter, req *http.Request, err error) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(err.Error()))
	}

	if s.rewriteURL {
		proxy := newReverseProxy(urlVal)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	} else {
		proxy := httputil.NewSingleHostReverseProxy(urlVal)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	}
}

// CreateProxyRoute create proxy route
func CreateProxyRoute(uriPattern, method, reallyURL string, rewriteURL bool) Route {
	return &proxyRoute{uriPattern: uriPattern, method: method, reallyURL: reallyURL, rewriteURL: rewriteURL}
}
