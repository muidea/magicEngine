package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/muidea/magicCommon/foundation/log"
)

// StaticOptions 是指定静态文件服务配置选项的结构体
type StaticOptions struct {
	RootPath string
	// PrefixUri 是用于提供静态目录内容的可选前缀
	PrefixUri string
	// SkipLogging 在提供静态文件时禁用 [Static] 日志消息
	SkipLogging bool
	// IndexFile 定义作为索引服务的文件（如果存在）
	IndexFile string
	// Expires 定义用于生成 HTTP Expires 头的用户自定义函数
	// https://developers.google.com/speed/docs/insights/LeverageBrowserCaching
	Expires func() string
	// Fallback 定义在找不到请求资源时提供默认 URL
	Fallback string
	// ExcludeUri 定义此处理器不应处理的 URL 模式
	ExcludeUri string
}

func prepareStaticOptions(option *StaticOptions) StaticOptions {
	opt := *option

	// 默认值
	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}
	// 标准化提供的前缀
	if opt.PrefixUri != "" {
		// 确保有前导 '/'
		if opt.PrefixUri[0] != '/' {
			opt.PrefixUri = "/" + opt.PrefixUri
		}
		// 移除任何尾随 '/'
		opt.PrefixUri = strings.TrimRight(opt.PrefixUri, "/")
	}
	return opt
}

// static 静态文件处理器
type static struct {
	rootPath     string
	subPrefixUri string
}

// MiddleWareHandle 处理静态文件请求的中间件
func (s *static) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	var err error
	staticVal := ctx.Context().Value(systemStatic{})
	if staticVal == nil {
		panicInfo("无法获取静态处理器")
	}

	defer func() {
		if err != nil {
			ctx.Next()
		}
	}()

	staticOpt := staticVal.(*StaticOptions)

	rootDirectory := staticOpt.RootPath
	if !filepath.IsAbs(rootDirectory) {
		rootDirectory = filepath.Join(s.rootPath, rootDirectory)
	}
	// 防止directory为相对路径
	if !filepath.IsAbs(rootDirectory) {
		rootDirectory = filepath.Join(Root, rootDirectory)
	}

	dir := http.Dir(rootDirectory)
	opt := prepareStaticOptions(staticOpt)

	// 检查HTTP方法是否为GET或HEAD
	if req.Method != GET && req.Method != HEAD {
		err = fmt.Errorf("no matching http method found")
		return
	}
	if opt.ExcludeUri != "" && strings.HasPrefix(req.URL.Path, opt.ExcludeUri) {
		err = fmt.Errorf("the requested url was not found on this server")
		return
	}

	fileUri := req.URL.Path

	// 如果有前缀，通过去掉前缀来过滤请求
	prefixUrl := filepath.Join(opt.PrefixUri, s.subPrefixUri)
	if prefixUrl != "" {
		if !strings.HasPrefix(fileUri, prefixUrl) {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
		fileUri = fileUri[len(opt.PrefixUri):]
		if fileUri != "" && fileUri[0] != '/' {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
	}

	staticFile, staticErr := dir.Open(fileUri)
	if staticErr != nil {
		// 在放弃之前尝试回退文件
		if opt.Fallback != "" {
			fileUri = opt.Fallback // 保持日志记录的真实性
			staticFile, staticErr = dir.Open(opt.Fallback)
		}

		if staticErr != nil {
			// 丢弃错误？
			err = staticErr
			return
		}
	}
	defer staticFile.Close()

	staticFileInfo, staticFileErr := staticFile.Stat()
	if staticFileErr != nil {
		err = staticFileErr
		return
	}

	// 尝试提供索引文件
	if staticFileInfo.IsDir() {
		// 如果缺少尾随斜杠则重定向
		if !strings.HasSuffix(req.URL.Path, "/") {
			dest := url.URL{
				Path:     req.URL.Path + "/",
				RawQuery: req.URL.RawQuery,
				Fragment: req.URL.Fragment,
			}
			http.Redirect(res, req, dest.String(), http.StatusFound)
			return
		}

		fileUri = path.Join(fileUri, opt.IndexFile)
		staticFile, staticFileErr = dir.Open(fileUri)
		if staticFileErr != nil {
			err = staticFileErr
			return
		}
		defer staticFile.Close()

		staticFileInfo, staticFileErr = staticFile.Stat()
		if staticFileErr != nil {
			err = staticFileErr
			return
		}
		if staticFileInfo.IsDir() {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
	}

	if !opt.SkipLogging {
		log.Infof("[Static] Serving " + fileUri)
	}

	// 为静态内容添加过期头
	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, fileUri, staticFileInfo.ModTime(), staticFile)
}

func StaticHandler(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	staticVal := ctx.Value(systemStatic{})
	if staticVal == nil {
		panicInfo("无法获取静态处理器")
	}

	staticOpt := staticVal.(*StaticOptions)

	rootDirectory := staticOpt.RootPath
	// 防止directory为相对路径
	if !filepath.IsAbs(rootDirectory) {
		rootDirectory = filepath.Join(Root, rootDirectory)
	}

	dir := http.Dir(rootDirectory)
	opt := prepareStaticOptions(staticOpt)
	uriFilePath := req.URL.Path

	var err error
	var staticFileHandle http.File

	staticUriFileHandle, staticUriFileErr := dir.Open(uriFilePath)
	defer func() {
		if staticUriFileHandle != nil {
			staticUriFileHandle.Close()
		}

		if err != nil {
			log.Warnf("[Static] Failed to serve %s: %v", uriFilePath, err)
			res.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if staticUriFileErr != nil {
		// 在放弃之前尝试回退文件
		if opt.Fallback != "" {
			uriFilePath = opt.Fallback // 保持日志记录的真实性
			staticUriFileHandle, staticUriFileErr = dir.Open(opt.Fallback)
		}

		if staticUriFileErr != nil {
			// 丢弃错误？
			err = staticUriFileErr
			return
		}
		staticFileHandle = staticUriFileHandle
	} else {
		staticFileHandle = staticUriFileHandle
	}

	staticFileInfo, staticFileErr := staticFileHandle.Stat()
	if staticFileErr != nil {
		err = staticFileErr
		return
	}

	// 尝试提供索引文件
	if staticFileInfo.IsDir() {
		if opt.IndexFile == "" {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}

		// 如果缺少尾随斜杠则重定向
		if !strings.HasSuffix(req.URL.Path, "/") {
			dest := url.URL{
				Path:     req.URL.Path + "/",
				RawQuery: req.URL.RawQuery,
				Fragment: req.URL.Fragment,
			}
			http.Redirect(res, req, dest.String(), http.StatusFound)
			return
		}

		uriFilePath = path.Join(uriFilePath, opt.IndexFile)
		staticIndexFileHandle, staticIndexFileErr := dir.Open(uriFilePath)
		if staticIndexFileErr != nil {
			err = staticIndexFileErr
			return
		}
		defer staticIndexFileHandle.Close()

		staticFileInfo, staticFileErr = staticIndexFileHandle.Stat()
		if staticFileErr != nil {
			err = staticFileErr
			return
		}
		if staticFileInfo.IsDir() {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}

		staticFileHandle = staticIndexFileHandle
	}

	// 为静态内容添加过期头
	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, uriFilePath, staticFileInfo.ModTime(), staticFileHandle)
}
