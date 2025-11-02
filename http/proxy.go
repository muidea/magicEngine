package http

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/muidea/magicCommon/foundation/log"
)

// newReverseProxy 创建一个新的反向代理，将请求转发到指定的目标URL
// 它会合并目标URL和传入请求中的查询参数
func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery, _ := url.ParseQuery(target.RawQuery)
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		reqQuery, _ := url.ParseQuery(req.URL.RawQuery)
		for k, v := range targetQuery {
			reqQuery.Set(k, v[0])
		}
		req.URL.RawQuery = reqQuery.Encode()
	}
	return &httputil.ReverseProxy{Director: director}
}

// proxyRoute 表示一个代理路由，用于将请求转发到目标URL
type proxyRoute struct {
	uriPattern string
	method     string
	targetURL  string
	rewriteURL bool
}

// Pattern 返回此代理路由的URI模式
func (s *proxyRoute) Pattern() string {
	return s.uriPattern
}

// Method 返回此代理路由的HTTP方法
func (s *proxyRoute) Method() string {
	return s.method
}

// Handler 返回路由处理函数
func (s *proxyRoute) Handler() RouteHandleFunc {
	return s.proxyFun
}

// proxyFun 是实际处理请求转发的函数
func (s *proxyRoute) proxyFun(_ context.Context, res http.ResponseWriter, req *http.Request) {
	// 解析目标URL
	targetUri, err := url.Parse(s.targetURL)
	if err != nil {
		log.Criticalf("illegal proxy target URL, url:%s", s.targetURL)
		return
	}

	// 获取目标URL和当前请求的查询参数
	targetQuery := targetUri.Query()
	reqQuery := req.URL.Query()

	// 检查是否存在动态路径标记和值，用于替换目标URL中的占位符
	dynamicTAG := req.Header.Get(DynamicTag)
	dynamicValue := req.Header.Get(DynamicValue)
	if dynamicTAG != "" && dynamicValue != "" {
		targetUri.Path = strings.ReplaceAll(targetUri.Path, dynamicTAG, dynamicValue)
	}

	// 将请求中的查询参数合并到目标URL参数中
	for k, v := range reqQuery {
		targetQuery.Set(k, v[0])
	}

	// 更新目标URL的查询参数
	targetUri.RawQuery = targetQuery.Encode()

	// 如果目标URL没有主机名，则执行重定向
	if targetUri.Hostname() == "" {
		http.Redirect(res, req, targetUri.String(), http.StatusSeeOther)
		return
	}

	// errorHandler 处理代理转发过程中的错误
	errorHandler := func(res http.ResponseWriter, req *http.Request, err error) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(err.Error()))
	}

	// 根据rewriteURL标志选择不同的代理方式
	if s.rewriteURL {
		// 使用自定义反向代理，支持URL重写
		proxy := newReverseProxy(targetUri)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	} else {
		// 使用标准单主机反向代理
		proxy := httputil.NewSingleHostReverseProxy(targetUri)
		proxy.ErrorHandler = errorHandler

		proxy.ServeHTTP(res, req)
	}
}

// CreateProxyRoute 创建代理路由
func CreateProxyRoute(uriPattern, method, targetURL string, rewriteURL bool) Route {
	return &proxyRoute{uriPattern: uriPattern, method: method, targetURL: targetURL, rewriteURL: rewriteURL}
}
