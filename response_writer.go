package magicengine

import (
	"net/http"
	"net/textproto"
)

// ResponseWriter is a wrapper around http.ResponseWriter that provides extra information about
// the response. It is recommended that middleware handlers use this construct to wrap a responsewriter
// if the functionality calls for it.
type ResponseWriter interface {
	http.ResponseWriter
	// Status returns the status code of the response or 0 if the response has not been written.
	Status() int
	// Written returns whether or not the ResponseWriter has been written.
	Written() bool
	// Size returns the size of the response body.
	Size() int
}

// NewResponseWriter creates a ResponseWriter that wraps an http.ResponseWriter
func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	newRw := responseWriter{rw, 0, 0}
	if cn, ok := rw.(http.CloseNotifier); ok {
		return &closeNotifyResponseWriter{newRw, cn}
	}
	return &newRw
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
		// The status will be StatusOK if WriteHeader has not been called yet
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

type closeNotifyResponseWriter struct {
	responseWriter
	closeNotifier http.CloseNotifier
}

func (rw *closeNotifyResponseWriter) CloseNotify() <-chan bool {
	return rw.closeNotifier.CloseNotify()
}
