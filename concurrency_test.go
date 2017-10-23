package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitGroup(t *testing.T) {
	var wg = NewWaitGroup()
	wg.Add(1)
	if wg.Aggregate() != 1 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate())
	}
	wg.Done()
	if wg.Aggregate() != 0 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate())
	}
}

func TestConcurrencyMiddleware(t *testing.T) {
	var wg = NewWaitGroup()
	var middleware = NewConcurrencyTrackingMiddleware(wg)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wg.Aggregate() != 1 {
			t.Fatalf("wrong internal count: %f", wg.Aggregate())
		}
	}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if wg.Aggregate() != 0 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate())
	}
}
