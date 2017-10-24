package loadshed

import (
	"context"
	"fmt"
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
		m.aggregators = append(m.aggregators, rolling.NewPercentageRollup(NewAvgCPU(pollingInterval, windowSize), lower, upper, "ChanceCPU"))
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
		m.aggregators = append(m.aggregators, rolling.NewPercentageRollup(wg, float64(lower), float64(upper), "ChanceConcurrency"))
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
		var a = rolling.NewLimitedRollup(requiredPoints, w, rolling.NewPercentageRollup(rolling.NewAverageRollup(w, "AverageLatency"), lower, upper, "ChanceAverageLatency"))
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
		var a = rolling.NewLimitedRollup(requiredPoints, w, rolling.NewPercentageRollup(rolling.NewPercentileRollup(percentile, w, preallocHint, fmt.Sprintf("P%fLatency", percentile)), lower, upper, fmt.Sprintf("ChanceP%fLatency", percentile)))
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
	random      func() float64
	callback    http.Handler
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var result *rolling.Aggregate
	for _, aggregator := range m.aggregators {
		var r = aggregator.Aggregate()
		if result == nil || r.Value > result.Value {
			result = r
		}
	}
	var chance = m.random()
	if chance < result.Value {
		m.callback.ServeHTTP(w, r.WithContext(NewContext(r.Context(), result)))
		return
	}
	m.next.ServeHTTP(w, r)
}

func defaultCallback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

var zeroAggregator = rolling.NewSumRollup(rolling.NewPointWindow(1), "Zero")

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
		return m
	}
}

type ctxKey string

var key = ctxKey("loadshed")

// NewContext inserts an aggregate into the context after a request has been
// rejected.
func NewContext(ctx context.Context, val *rolling.Aggregate) context.Context {
	return context.WithValue(ctx, key, val)
}

// FromContext extracts an aggregate from the context after a request has
// been rejected.
func FromContext(ctx context.Context) *rolling.Aggregate {
	if v, ok := ctx.Value(key).(*rolling.Aggregate); ok {
		return v
	}
	return nil
}
