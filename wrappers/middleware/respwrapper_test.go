package middleware

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"reflect"
	"testing"
)

type fixtureResponseWriter struct {
	calledHeader      bool
	calledWrite       bool
	calledWriteHeader bool
}

func (r *fixtureResponseWriter) Header() http.Header {
	r.calledHeader = true
	return http.Header{}
}
func (r *fixtureResponseWriter) Write(b []byte) (int, error) {
	r.calledWrite = true
	return len(b), nil
}
func (r *fixtureResponseWriter) WriteHeader(int) {
	r.calledWriteHeader = true
}

type fixtureCloseNotifier struct {
	fixtureResponseWriter
	calledCloseNotify bool
}

func (r *fixtureCloseNotifier) CloseNotify() <-chan bool {
	r.calledCloseNotify = true
	return make(<-chan bool)
}

type fixtureFlusher struct {
	fixtureResponseWriter
	calledFlush bool
}

func (r *fixtureFlusher) Flush() {
	r.calledFlush = true
}

type fixtureHijacker struct {
	fixtureResponseWriter
	calledHijack bool
}

func (r *fixtureHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.calledHijack = true
	return nil, nil, nil
}

type fixtureHTTPResponseWriter struct {
	fixtureResponseWriter
	fixtureHijacker
	fixtureFlusher
	fixtureCloseNotifier
	calledReadFrom bool
}

func (r *fixtureHTTPResponseWriter) ReadFrom(reader io.Reader) (n int64, err error) {
	r.calledReadFrom = true
	return 0, nil
}

func TestRespWrapperInterfaces(t *testing.T) {
	var _ http.CloseNotifier = &fancyWriter{} // nolint
	var _ http.Flusher = &fancyWriter{}
	var _ http.Hijacker = &fancyWriter{}
	var _ io.ReaderFrom = &fancyWriter{}
	var _ http.Flusher = &flushWriter{}
}

func TestSimpleImplementationOnlyGetsBasic(t *testing.T) {
	var r = fixtureResponseWriter{}
	var result = wrapWriter(&r)

	if _, ok := result.(http.ResponseWriter); !ok {
		t.Fatal("Did not get a ResponseWriter back.")
	}
	if _, ok := result.(*basicWriter); !ok {
		t.Fatalf("Did not wrap a simple ResponseWriter with a basicWiter. %s", reflect.TypeOf(result))
	}
}

func TestFlusherOnlyGetsFlusher(t *testing.T) {
	var r = fixtureFlusher{fixtureResponseWriter{}, false}
	var result = wrapWriter(&r)

	if _, ok := result.(http.ResponseWriter); !ok {
		t.Fatal("Did not get a ResponseWriter back.")
	}
	if _, ok := result.(http.Flusher); !ok {
		t.Fatal("Did not get a flusher back.")
	}
	if _, ok := result.(*flushWriter); !ok {
		t.Fatalf("Did not wrap a flusher with a flushWiter. %s", reflect.TypeOf(result))
	}
}

func TestFullFeaturedGetsFancy(t *testing.T) {
	var base = fixtureResponseWriter{}
	var r = fixtureHTTPResponseWriter{
		base,
		fixtureHijacker{base, false},
		fixtureFlusher{base, false},
		fixtureCloseNotifier{base, false},
		false,
	}
	var result = wrapWriter(&r)

	if _, ok := result.(http.ResponseWriter); !ok {
		t.Fatal("Did not get a ResponseWriter back.")
	}
	if _, ok := result.(http.CloseNotifier); !ok { // nolint
		t.Fatal("Did not get a CloseNotifier back.")
	}
	if _, ok := result.(http.Hijacker); !ok {
		t.Fatal("Did not get a Hijacker back.")
	}
	if _, ok := result.(io.ReaderFrom); !ok {
		t.Fatal("Did not get a ReaderFrom back.")
	}
	if _, ok := result.(*fancyWriter); !ok {
		t.Fatalf("Did not wrap a full implementation with a fancyWriter. %s", reflect.TypeOf(result))
	}
}

func TestBasicWriterCallsWriteHeaderOnce(t *testing.T) {
	var wrapped = &fixtureResponseWriter{}
	var r = basicWriter{ResponseWriter: wrapped}

	r.WriteHeader(9)
	r.WriteHeader(10)
	if r.Status() != 9 {
		t.Fatalf("Expected code to be 9 but got %d", r.Status())
	}
	if !wrapped.calledWriteHeader {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestBasicWriterCanDefaultResponseCode(t *testing.T) {
	var wrapped = &fixtureResponseWriter{}
	var r = basicWriter{ResponseWriter: wrapped}

	r.maybeWriteHeader()
	if r.code != 200 {
		t.Fatalf("Expected code to be 200 but got %d", r.code)
	}
	if !wrapped.calledWriteHeader {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestBasicWriterDoesNotOverWriteWithDefaultCode(t *testing.T) {
	var wrapped = &fixtureResponseWriter{}
	var r = basicWriter{ResponseWriter: wrapped}

	r.WriteHeader(9)
	r.maybeWriteHeader()
	if r.Status() != 9 {
		t.Fatalf("Expected code to be 9 but got %d", r.Status())
	}
	if !wrapped.calledWriteHeader {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestBasicWriterCountsBytesWritten(t *testing.T) {
	var wrapped = &fixtureResponseWriter{}
	var r = basicWriter{ResponseWriter: wrapped}

	_, _ = r.Write([]byte(`TEST`))
	if r.BytesWritten() != 4 {
		t.Fatalf("Expected 4 bytes written. Got %d", r.BytesWritten())
	}
	if !wrapped.calledWrite {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
	_, _ = r.Write([]byte(`TEST`))
	if r.BytesWritten() != 8 {
		t.Fatalf("Expected 8 bytes written. Got %d", r.BytesWritten())
	}
}

func TestFancyWriterCloseNotify(t *testing.T) {
	var base = fixtureResponseWriter{}
	var wrapped = fixtureHTTPResponseWriter{
		base,
		fixtureHijacker{base, false},
		fixtureFlusher{base, false},
		fixtureCloseNotifier{base, false},
		false,
	}
	var r = wrapWriter(&wrapped)

	_ = r.(http.CloseNotifier).CloseNotify() // nolint
	if !wrapped.calledCloseNotify {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestFancyWriterHijacker(t *testing.T) {
	var base = fixtureResponseWriter{}
	var wrapped = fixtureHTTPResponseWriter{
		base,
		fixtureHijacker{base, false},
		fixtureFlusher{base, false},
		fixtureCloseNotifier{base, false},
		false,
	}
	var r = wrapWriter(&wrapped)

	_, _, _ = r.(http.Hijacker).Hijack()
	if !wrapped.calledHijack {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestFancyWriterFlush(t *testing.T) {
	var base = fixtureResponseWriter{}
	var wrapped = fixtureHTTPResponseWriter{
		base,
		fixtureHijacker{base, false},
		fixtureFlusher{base, false},
		fixtureCloseNotifier{base, false},
		false,
	}
	var r = wrapWriter(&wrapped)

	r.(http.Flusher).Flush()
	if !wrapped.calledFlush {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}

func TestFancyWriterReaderFrom(t *testing.T) {
	var base = fixtureResponseWriter{}
	var wrapped = fixtureHTTPResponseWriter{
		base,
		fixtureHijacker{base, false},
		fixtureFlusher{base, false},
		fixtureCloseNotifier{base, false},
		false,
	}
	var r = wrapWriter(&wrapped)

	_, _ = r.(io.ReaderFrom).ReadFrom(bytes.NewBufferString(`TEST`))
	if !wrapped.calledReadFrom {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
	if !wrapped.calledWriteHeader {
		t.Fatal("Wrapper did not set header before writing data.")
	}
}

func TestFlushWriterFlush(t *testing.T) {
	var wrapped = fixtureFlusher{fixtureResponseWriter{}, false}
	var r = wrapWriter(&wrapped)

	r.(http.Flusher).Flush()
	if !wrapped.calledFlush {
		t.Fatal("Wrapper did not called wrapped implementation.")
	}
}
