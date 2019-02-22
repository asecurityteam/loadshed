package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/asecurityteam/loadshed"
)

func TestMiddleware(t *testing.T) {
	var l = &fakeLoadShedder{}
	var middleware = New(l)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestMiddlewareError(t *testing.T) {
	var l = &fakeLoadShedder{err: fmt.Errorf("")}
	var middleware = New(l)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestMiddlewareRejectedError(t *testing.T) {
	var l = &fakeLoadShedder{err: loadshed.Rejected{}}
	var middleware = New(l)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestMiddlewareCallback(t *testing.T) {
	var l = &fakeLoadShedder{err: loadshed.Rejected{}}
	var cb = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}
	var middleware = New(l, Callback(http.HandlerFunc(cb)))
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTeapot {
		t.Fatalf("middleware did not call default handler: %d", w.Code)
	}

}

func TestMiddlewareErrorCode(t *testing.T) {
	var l = &fakeLoadShedder{}
	var errCode = ErrCodes([]int{http.StatusInternalServerError})
	var middleware = New(l, errCode)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestMiddlewareErrorCodeAndError(t *testing.T) {
	var l = &fakeLoadShedder{err: loadshed.Rejected{}}
	var errCode = ErrCodes([]int{http.StatusInternalServerError})
	var middleware = New(l, errCode)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

type fakeLoadShedder struct {
	Counter int32
	err     error
}

func (f *fakeLoadShedder) Do(run func() error) error {
	atomic.AddInt32(&f.Counter, 1)
	if f.err != nil {
		return f.err
	}
	return run()

}
