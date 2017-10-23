package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/atlassian/rolling"
)

func TestLatencyMiddleware(t *testing.T) {
	var window = rolling.NewPointWindow(1)
	var middleware = NewLatencyTrackingMiddleware(window)
	var handler = middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
	}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var a = rolling.NewSumAggregator(window)
	var result = a.Aggregate()
	if result < (5*time.Millisecond).Seconds() || result > (6*time.Millisecond).Seconds() {
		t.Fatalf("incorrect latency record: %f", result)
	}
}
