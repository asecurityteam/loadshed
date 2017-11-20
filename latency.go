package loadshed

import (
	"time"

	"bitbucket.org/atlassian/rolling"
)

type latencyDecorator struct {
	feeder rolling.Feeder
}

func (h *latencyDecorator) Wrap(next func() error) func() error {
	return func() error {
		var start = time.Now()
		var e error
		e = next()
		h.feeder.Feed(time.Since(start).Seconds())
		return e
	}
}

// newLatencyTrackingDecorator tracks latencies of an acion using a given
// rollingdwindow.Feeder.
func newLatencyTrackingDecorator(feeder rolling.Feeder) wrapper {
	return &latencyDecorator{feeder}
}
