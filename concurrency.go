package loadshed

import (
	"net/http"
	"sync"
	"sync/atomic"
)

// WaitGroup wraps a sync.WaitGroup to make it usable as a load shedding tool.
type WaitGroup struct {
	*sync.WaitGroup
	concurrent *int32
}

// NewWaitGroup generates a specialised WaitGroup that tracks the number of
// concurrent operations. This implementation also satisfies the Aggregator
// interface from bitbucket.org/atlassian/rolling so that this can be fed into
// a calculation of system health.
func NewWaitGroup() *WaitGroup {
	var w = &WaitGroup{
		WaitGroup:  &sync.WaitGroup{},
		concurrent: new(int32),
	}
	return w
}

// Aggregate returns the current concurrency value
func (c *WaitGroup) Aggregate() float64 {
	return float64(atomic.LoadInt32(c.concurrent))
}

// Add some number of concurrent operations.
func (c *WaitGroup) Add(delta int) {
	c.WaitGroup.Add(delta)
	atomic.AddInt32(c.concurrent, int32(delta))
}

// Done markes an operation as complete and removes the tracking.
func (c *WaitGroup) Done() {
	c.WaitGroup.Done()
	atomic.AddInt32(c.concurrent, -1)
}

// Wait for all operations to complete.
func (c *WaitGroup) Wait() {
	c.WaitGroup.Wait()
}

type concurrencyMiddleware struct {
	next http.Handler
	wg   *WaitGroup
}

func (h *concurrencyMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.wg.Add(1)
	defer h.wg.Done()
	h.next.ServeHTTP(w, r)
}

// NewConcurrencyTrackingMiddleware tracks concurrent HTTP requests using the
// given WaitGroup.
func NewConcurrencyTrackingMiddleware(wg *WaitGroup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &concurrencyMiddleware{next, wg}
	}
}
