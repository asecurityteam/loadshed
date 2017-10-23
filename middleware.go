package loadshed

import (
	"math/rand"
	"net/http"
	"time"

	"bitbucket.org/atlassian/rolling"
)

// Option modifies the behaviour of the load shedding Middleware.
type Option func(*Middleware) *Middleware

// CPU generates an option that adds a rolling average of CPU usage to the
// load shedding calculation. It will configure the Middleware to reject a
// percentage of traffic once the average CPU usage is between lower and upper.
func CPU(lower float64, upper float64, pollingInterval time.Duration, windowSize int) Option {
	return func(m *Middleware) *Middleware {
		m.aggregators = append(m.aggregators, rolling.NewPercentageAggregator(NewAvgCPU(pollingInterval, windowSize), lower, upper))
		return m
	}
}

// Concurrency generates an option that adds total concurrent requests to the
// load shedding calculation. Once the requests in flight reaches a value
// between lower and upper the Middleware will begin rejecting new requests
// based on the distance between the threshold values.
func Concurrency(lower int, upper int, wg *WaitGroup) Option {
	return func(m *Middleware) *Middleware {
		if wg == nil {
			wg = NewWaitGroup()
		}
		m.aggregators = append(m.aggregators, rolling.NewPercentageAggregator(wg, float64(lower), float64(upper)))
		m.next = NewConcurrencyTrackingMiddleware(wg)(m.next)
		return m
	}
}

const defaultHint = 2500

// AverageLatency generates an option that adds average request latency within
// a rolling time window to the load shedding calculation. If the average value,
// in seconds, falls between lower and upper then a percentage of new requests
// will be rejected. The rolling window is configured by defining a bucket size
// and number of buckets. The preallocHint is an optimisation for keeping the
// number of alloc calls low. If the hint is zero then a default value is
// used.
func AverageLatency(lower float64, upper float64, bucketSize time.Duration, buckets int, preallocHint int, requiredPoints int) Option {
	return func(m *Middleware) *Middleware {
		if preallocHint < 1 {
			preallocHint = defaultHint
		}
		var w = rolling.NewTimeWindow(bucketSize, buckets, preallocHint)
		var a = rolling.NewLimitedAggregator(requiredPoints, w, rolling.NewAverageAggregator(w))
		m.aggregators = append(m.aggregators, a)
		m.next = NewLatencyTrackingMiddleware(w)(m.next)
		return m
	}
}

// PercentileLatency generates an option much like AverageLatency except the
// aggregation is computed as a percentile of the data recorded rather than an
// average. The percentile should be given as N%. For example, 95.0 or 99.0.
// Fractional percentiles, like 99.9, are also valid.
func PercentileLatency(lower float64, upper float64, bucketSize time.Duration, buckets int, preallocHint int, requiredPoints int, percentile float64) Option {
	return func(m *Middleware) *Middleware {
		if preallocHint < 1 {
			preallocHint = defaultHint
		}
		var w = rolling.NewTimeWindow(bucketSize, buckets, preallocHint)
		var a = rolling.NewLimitedAggregator(requiredPoints, w, rolling.NewPercentileAggregator(percentile, w, preallocHint))
		m.aggregators = append(m.aggregators, a)
		m.next = NewLatencyTrackingMiddleware(w)(m.next)
		return m
	}
}

// Callback generates an option that sets the Handler that will be executed
// whenever a request is rejected due to the load shedding Middleware.
func Callback(h http.Handler) Option {
	return func(m *Middleware) *Middleware {
		m.callback = h
		return m
	}
}

// Aggregator adds an arbitrary Aggregator to the evaluation for load shedding.
// The result of the aggregator will be interpreted as a percentage value
// between 0.0 and 1.0. This value will be used as the percentage of requests
// to reject.
func Aggregator(a rolling.Aggregator) Option {
	return func(m *Middleware) *Middleware {
		m.aggregators = append(m.aggregators, a)
		return m
	}
}

// Middleware is an http.Handler wrapper that rejects a percentage of requests
// based on aggregation of system load data.
type Middleware struct {
	next        http.Handler
	aggregators []rolling.Aggregator
	chain       []func(http.Handler) http.Handler
	aggregator  rolling.Aggregator
	random      func() float64
	callback    http.Handler
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var evaluation = m.aggregator.Aggregate()
	var chance = m.random()
	if chance < evaluation {
		m.callback.ServeHTTP(w, r)
		return
	}
	m.next.ServeHTTP(w, r)
}

func defaultCallback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

var zeroAggregator = rolling.NewSumAggregator(rolling.NewPointWindow(1))

// NewMiddlware generates an http.Handler wrapper that sheds load based
// on some definition of system load.
func NewMiddlware(options ...Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var r = rand.New(rand.NewSource(time.Now().UnixNano()))
		var m = &Middleware{next: next, random: r.Float64, callback: http.HandlerFunc(defaultCallback)}
		for _, option := range options {
			m = option(m)
		}
		if len(m.aggregators) < 1 {
			m.aggregators = append(m.aggregators, zeroAggregator)
		}
		m.aggregator = rolling.NewMaxAggregator(rolling.NewAggregatorIterator(m.aggregators...))
		return m
	}
}
