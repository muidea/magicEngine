package http

import (
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
	Path string
	// Prefix 是用于提供静态目录内容的可选前缀
	Prefix string
	// SkipLogging 在提供静态文件时禁用 [Static] 日志消息
	SkipLogging bool
	// IndexFile 定义作为索引服务的文件（如果存在）
	IndexFile string
	// Expires 定义用于生成 HTTP Expires 头的用户自定义函数
	// https://developers.google.com/speed/docs/insights/LeverageBrowserCaching
	Expires func() string
	// Fallback 定义在找不到请求资源时提供默认 URL
	Fallback string
	// Exclude 定义此处理器不应处理的 URL 模式
	Exclude string
}

func prepareStaticOptions(option *StaticOptions) StaticOptions {
	opt := *option

	// 默认值
	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}
	// 标准化提供的前缀
	if opt.Prefix != "" {
		// 确保有前导 '/'
		if opt.Prefix[0] != '/' {
			opt.Prefix = "/" + opt.Prefix
		}
		// 移除任何尾随 '/'
		opt.Prefix = strings.TrimRight(opt.Prefix, "/")
	}
	return opt
}

// static 静态文件处理器
type static struct {
	rootPath string
}

// MiddleWareHandle 处理静态文件请求的中间件
func (s *static) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	var err error
	staticObj := ctx.Context().Value(systemStatic{})
	if staticObj == nil {
		panicInfo("无法获取静态处理器")
	}

	defer func() {
		if err != nil {
			ctx.Next()
		}
	}()

	staticOpt := staticObj.(*StaticOptions)

	directory := staticOpt.Path
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(s.rootPath, directory)
	}
	// 防止directory为相对路径
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(Root, directory)
	}

	dir := http.Dir(directory)
	opt := prepareStaticOptions(staticOpt)

	// 检查HTTP方法是否为GET或HEAD
	if req.Method != "GET" && req.Method != "HEAD" {
		err = fmt.Errorf("no matching http method found")
		return
	}
	if opt.Exclude != "" && strings.HasPrefix(req.URL.Path, opt.Exclude) {
		err = fmt.Errorf("the requested url was not found on this server")
		return
	}

	file := req.URL.Path

	// 如果有前缀，通过去掉前缀来过滤请求
	if opt.Prefix != "" {
		if !strings.HasPrefix(file, opt.Prefix) {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
		file = file[len(opt.Prefix):]
		if file != "" && file[0] != '/' {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
	}

	staticFile, staticErr := dir.Open(file)
	if staticErr != nil {
		// 在放弃之前尝试回退文件
		if opt.Fallback != "" {
			file = opt.Fallback // 保持日志记录的真实性
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

		file = path.Join(file, opt.IndexFile)
		staticFile, staticFileErr = dir.Open(file)
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
		log.Infof("[Static] Serving " + file)
	}

	// 为静态内容添加过期头
	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, file, staticFileInfo.ModTime(), staticFile)
}