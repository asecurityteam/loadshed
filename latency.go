package loadshed

import (
	"net/http"
	"time"

	"bitbucket.org/atlassian/rolling"
)

type latencyMiddleware struct {
	feeder rolling.Feeder
	next   http.Handler
}

func (h *latencyMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var start = time.Now()
	h.next.ServeHTTP(w, r)
	h.feeder.Feed(time.Since(start).Seconds())
}

// NewLatencyTrackingMiddleware tracks latencies of HTTP requests using a given
// rollingdwindow.Feeder.
func NewLatencyTrackingMiddleware(feeder rolling.Feeder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &latencyMiddleware{feeder, next}
	}
}
