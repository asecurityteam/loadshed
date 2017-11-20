package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// writerProxy is a proxy around an http.ResponseWriter that allows you to hook
// into various parts of the response process.
// Code originally sources from https://github.com/pressly/chi/blob/master/middleware/logger.go.
type writerProxy interface {
	http.ResponseWriter
	// Status returns the HTTP status of the request, or 0 if one has not
	// yet been sent.
	Status() int
	// BytesWritten returns the total number of bytes sent to the client.
	BytesWritten() int
}

// wrapWriter wraps an http.ResponseWriter, returning a proxy that allows you to
// hook into various parts of the response process.
func wrapWriter(w http.ResponseWriter) writerProxy {
	var _, cn = w.(http.CloseNotifier)
	var _, fl = w.(http.Flusher)
	var _, hj = w.(http.Hijacker)
	var _, rf = w.(io.ReaderFrom)

	var bw = basicWriter{ResponseWriter: w}
	if cn && fl && hj && rf {
		return &fancyWriter{&bw}
	}
	if fl {
		return &flushWriter{&bw}
	}
	return &bw
}

// basicWriter wraps a http.ResponseWriter that implements the minimal
// http.ResponseWriter interface.
type basicWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
}

func (b *basicWriter) WriteHeader(code int) {
	if b.wroteHeader {
		return
	}
	b.code = code
	b.wroteHeader = true
	b.ResponseWriter.WriteHeader(code)
}

func (b *basicWriter) Write(buf []byte) (int, error) {
	b.WriteHeader(http.StatusOK)
	var n, err = b.ResponseWriter.Write(buf)
	b.bytes += n
	return n, err
}

func (b *basicWriter) maybeWriteHeader() {
	if b.wroteHeader {
		return
	}
	b.WriteHeader(http.StatusOK)
}

func (b *basicWriter) Status() int {
	return b.code
}

func (b *basicWriter) BytesWritten() int {
	return b.bytes
}

// fancyWriter is a writer that additionally satisfies http.CloseNotifier,
// http.Flusher, http.Hijacker, and io.ReaderFrom. It exists for the common case
// of wrapping the http.ResponseWriter that package http gives you, in order to
// make the proxied object support the full method set of the proxied object.
type fancyWriter struct {
	*basicWriter
}

func (f *fancyWriter) CloseNotify() <-chan bool {
	return f.basicWriter.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (f *fancyWriter) Flush() {
	f.basicWriter.ResponseWriter.(http.Flusher).Flush()
}

func (f *fancyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.basicWriter.ResponseWriter.(http.Hijacker).Hijack()
}

func (f *fancyWriter) ReadFrom(r io.Reader) (int64, error) {
	var rf = f.basicWriter.ResponseWriter.(io.ReaderFrom)
	f.basicWriter.maybeWriteHeader()
	return rf.ReadFrom(r)
}

type flushWriter struct {
	*basicWriter
}

func (f *flushWriter) Flush() {
	f.basicWriter.ResponseWriter.(http.Flusher).Flush()
}
