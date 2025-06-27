package http

import (
	"net/http"
	"net/textproto"
)

type ResponseWriter interface {
	http.ResponseWriter
	Status() int
	Written() bool
	Size() int
}

func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	return &responseWriter{rw, 0, 0}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

var contentType = textproto.CanonicalMIMEHeaderKey("content-type")

func (rw *responseWriter) verifyContentType() {
	contentVal := rw.Header().Get(contentType)
	if contentVal != "" {
		return
	}
	rw.Header().Set(contentType, "application/json; charset=utf-8")
}

func (rw *responseWriter) WriteHeader(s int) {
	rw.ResponseWriter.WriteHeader(s)
	rw.status = s
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.verifyContentType()

	if !rw.Written() {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) Size() int {
	return rw.size
}

func (rw *responseWriter) Written() bool {
	return rw.status != 0
}

func (rw *responseWriter) Flush() {
	rw.ResponseWriter.(http.Flusher).Flush()
}
