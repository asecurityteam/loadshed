package loadshed

import (
	"sync"
	"sync/atomic"

	"github.com/asecurityteam/rolling"
)

// WaitGroup wraps a sync.WaitGroup to make it usable as a load shedding tool.
type WaitGroup struct {
	*sync.WaitGroup
	concurrent *int32
}

// NewWaitGroup generates a specialised WaitGroup that tracks the number of
// concurrent operations. This implementation also satisfies the Aggregator
// interface from github.com/asecurityteam/rolling so that this can be fed into
// a calculation of system health.
func NewWaitGroup() *WaitGroup {
	var w = &WaitGroup{
		WaitGroup:  &sync.WaitGroup{},
		concurrent: new(int32),
	}
	return w
}

// Aggregate returns the current concurrency value
func (c *WaitGroup) Aggregate() *rolling.Aggregate {
	return &rolling.Aggregate{
		Source: nil,
		Name:   "WaitGroup",
		Value:  float64(atomic.LoadInt32(c.concurrent)),
	}
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

type concurrencyDecorator struct {
	wg *WaitGroup
}

func (h *concurrencyDecorator) Wrap(next func() error) func() error {
	return func() error {
		h.wg.Add(1)
		defer h.wg.Done()
		return next()
	}
}

// newConcurrencyTrackingDecorator tracks concurrent actions using the
// given WaitGroup.
func newConcurrencyTrackingDecorator(wg *WaitGroup) wrapper {
	return &concurrencyDecorator{wg}
}
