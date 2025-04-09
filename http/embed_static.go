package http

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/log"
)

type EmbedFile struct {
	ModTime     time.Time
	FileContext []byte
}

type EmbedStatic struct {
	embedPath      string
	prefixPath     string
	templateFS     embed.FS
	staticFileInfo sync.Map
}

func NewEmbedStatic(templateFS embed.FS, embedPath, prefixPath string) *EmbedStatic {
	return &EmbedStatic{
		embedPath:      embedPath,
		prefixPath:     prefixPath,
		templateFS:     templateFS,
		staticFileInfo: sync.Map{},
	}
}

func (s *EmbedStatic) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			ctx.Next()
		}
	}()

	if !strings.HasPrefix(req.URL.Path, s.prefixPath) {
		ctx.Next()
		return
	}

	if req.Method != "GET" && req.Method != "HEAD" {
		err = fmt.Errorf("no matching http method found")
		log.Errorf("static middleware, url:%s, error: %v", req.URL.Path, err)
		return
	}

	filePath := s.validatePath(req.URL.Path)
	staticFile, staticModTime, staticErr := s.findEmbedFile(filePath)
	if staticErr != nil {
		err = fmt.Errorf("find embed file failed, filePath:%s, error: %v", filePath, staticErr)
		log.Errorf("static middleware, url:%s, error: %v", req.URL.Path, err)
		return
	}

	http.ServeContent(res, req, filePath, staticModTime, staticFile)
}

func (s *EmbedStatic) validatePath(filePath string) (ret string) {
	if filePath == "" {
		filePath = "/"
	}

	if s.isDir(filePath) {
		filePath = path.Join(filePath, "index.html")
	}

	ret = path.Join(s.embedPath, filePath)
	return
}

func (s *EmbedStatic) isDir(pathVal string) bool {
	return len(pathVal) > 0 && pathVal[len(pathVal)-1] == '/'
}

func (s *EmbedStatic) findEmbedFile(filePath string) (content io.ReadSeeker, modTime time.Time, err error) {
	fileInfo, fileOK := s.staticFileInfo.Load(filePath)
	if fileOK {
		content = bytes.NewReader(fileInfo.(EmbedFile).FileContext)
		modTime = fileInfo.(EmbedFile).ModTime
		return
	}

	fsInfo, fsErr := fs.Stat(s.templateFS, filePath)
	if fsErr != nil {
		err = fsErr
		return
	}
	contentVal, contentErr := fs.ReadFile(s.templateFS, filePath)
	if contentErr != nil {
		err = contentErr
		return
	}
	s.staticFileInfo.Store(filePath, EmbedFile{
		ModTime:     fsInfo.ModTime(),
		FileContext: contentVal,
	})

	content = bytes.NewReader(contentVal)
	modTime = fsInfo.ModTime()
	return
}
