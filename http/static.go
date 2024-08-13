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

// StaticOptions is a struct for specifying configuration options for the martini.Static middleware.
type StaticOptions struct {
	Path string
	// Prefix is the optional prefix used to serve the static directory content
	Prefix string
	// SkipLogging will disable [Static] log messages when a static file is served.
	SkipLogging bool
	// IndexFile defines which file to serve as index if it exists.
	IndexFile string
	// Expires defines which user-defined function to use for producing a HTTP Expires Header
	// https://developers.google.com/speed/docs/insights/LeverageBrowserCaching
	Expires func() string
	// Fallback defines a default URL to serve when the requested resource was
	// not found.
	Fallback string
	// Exclude defines a pattern for URLs this handler should never process.
	Exclude string
}

func prepareStaticOptions(option *StaticOptions) StaticOptions {
	opt := *option

	// Defaults
	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}
	// Normalize the prefix if provided
	if opt.Prefix != "" {
		// Ensure we have a leading '/'
		if opt.Prefix[0] != '/' {
			opt.Prefix = "/" + opt.Prefix
		}
		// Remove any trailing '/'
		opt.Prefix = strings.TrimRight(opt.Prefix, "/")
	}
	return opt
}

type static struct {
	rootPath string
}

// Static returns a middleware handler that serves static files in the given directory.
func (s *static) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	var err error
	staticObj := ctx.Context().Value(systemStatic)
	if staticObj == nil {
		panicInfo("cant\\'t get static handler")
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

	if req.Method != "GET" && req.Method != "HEAD" {
		err = fmt.Errorf("no matching http method found")
		return
	}
	if opt.Exclude != "" && strings.HasPrefix(req.URL.Path, opt.Exclude) {
		err = fmt.Errorf("the requested url was not found on this server")
		return
	}

	file := req.URL.Path

	// if we have a prefix, filter requests by stripping the prefix
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
		// try any fallback before giving up
		if opt.Fallback != "" {
			file = opt.Fallback // so that logging stays true
			staticFile, staticErr = dir.Open(opt.Fallback)
		}

		if staticErr != nil {
			// discard the error?
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

	// try to serve index file
	if staticFileInfo.IsDir() {
		// redirect if missing trailing slash
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

	// Add an Expires header to the static content
	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, file, staticFileInfo.ModTime(), staticFile)
}
