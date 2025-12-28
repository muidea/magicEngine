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

type Verifier interface {
	Verify(ctx RequestContext, res http.ResponseWriter, req *http.Request) error
}

// StaticOptions 是指定静态文件服务配置选项的结构体
type StaticOptions struct {
	Path string
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

func NewShareStatic(rootPath string) *ShareStatic {
	return &ShareStatic{
		rootPath:  rootPath,
		subPrefix: share,
	}
}

func NewPrivateStatic(rootPath string, verifier Verifier) *PrivateStatic {
	return &PrivateStatic{
		ShareStatic: ShareStatic{
			rootPath:  rootPath,
			subPrefix: private,
		},
		verifier: verifier,
	}
}

// ShareStatic 静态文件处理器
type ShareStatic struct {
	rootPath  string
	subPrefix string
}

// MiddleWareHandle 处理静态文件请求的中间件
func (s *ShareStatic) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
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

	rootDirectory := staticOpt.Path
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

	fileUrI := req.URL.Path

	// 如果有前缀，通过去掉前缀来过滤请求
	prefixUrl := filepath.Join(opt.PrefixUri, s.subPrefix)
	if prefixUrl != "" {
		if !strings.HasPrefix(fileUrI, prefixUrl) {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
		fileUrI = fileUrI[len(opt.PrefixUri):]
		if fileUrI != "" && fileUrI[0] != '/' {
			err = fmt.Errorf("the requested url was not found on this server")
			return
		}
	}

	staticFile, staticErr := dir.Open(fileUrI)
	if staticErr != nil {
		// 在放弃之前尝试回退文件
		if opt.Fallback != "" {
			fileUrI = opt.Fallback // 保持日志记录的真实性
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

		fileUrI = path.Join(fileUrI, opt.IndexFile)
		staticFile, staticFileErr = dir.Open(fileUrI)
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
		log.Infof("[Static] Serving " + fileUrI)
	}

	// 为静态内容添加过期头
	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, fileUrI, staticFileInfo.ModTime(), staticFile)
}

type PrivateStatic struct {
	ShareStatic

	verifier Verifier
}

func (s *PrivateStatic) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	if s.verifier != nil {
		if err := s.verifier.Verify(ctx, res, req); err != nil {
			res.WriteHeader(http.StatusForbidden)
			return
		}
	}

	s.ShareStatic.MiddleWareHandle(ctx, res, req)
}
