package loadshed

import (
	"fmt"
	"math/rand"
	"time"

	"bitbucket.org/atlassian/rolling"
)

// Doer is an interface representing load shedding interface with Do method
type Doer interface {
	Do(func() error) error
}

// wrapper is an interface representing the loadshed feeders
type wrapper interface {
	Wrap(func() error) func() error
}

// Option is a partial initializer for Loadshed
type Option func(*Loadshed) *Loadshed

const defaultHint = 1000

// PercentileLatency generates an option much like AverageLatency except the
// aggregation is computed as a percentile of the data recorded rather than an
// average. The percentile should be given as N%. For example, 95.0 or 99.0.
// Fractional percentiles, like 99.9, are also valid.
func PercentileLatency(lower float64, upper float64, bucketSize time.Duration, buckets int, preallocHint int, requiredPoints int, percentile float64) Option {
	return func(m *Loadshed) *Loadshed {
		if preallocHint < 1 {
			preallocHint = defaultHint
		}
		var w = rolling.NewTimeWindow(bucketSize, buckets, preallocHint)
		var a = rolling.NewLimitedRollup(requiredPoints, w, rolling.NewPercentageRollup(rolling.NewPercentileRollup(percentile, w, preallocHint, fmt.Sprintf("P%fLatency", percentile)), lower, upper, fmt.Sprintf("ChanceP%fLatency", percentile)))
		m.aggregators = append(m.aggregators, a)
		m.chain = append(m.chain, newLatencyTrackingDecorator(w).Wrap)
		return m
	}
}

// AverageLatency generates an option that adds average request latency within
// a rolling time window to the load shedding calculation. If the average value,
// in seconds, falls between lower and upper then a percentage of new requests
// will be rejected. The rolling window is configured by defining a bucket size
// and number of buckets. The preallocHint is an optimisation for keeping the
// number of alloc calls low. If the hint is zero then a default value is
// used.
func AverageLatency(lower float64, upper float64, bucketSize time.Duration, buckets int, preallocHint int, requiredPoints int) Option {
	return func(m *Loadshed) *Loadshed {
		if preallocHint < 1 {
			preallocHint = defaultHint
		}
		var w = rolling.NewTimeWindow(bucketSize, buckets, preallocHint)
		var a = rolling.NewLimitedRollup(requiredPoints, w, rolling.NewPercentageRollup(rolling.NewAverageRollup(w, "AverageLatency"), lower, upper, "ChanceAverageLatency"))
		m.aggregators = append(m.aggregators, a)
		m.chain = append(m.chain, newLatencyTrackingDecorator(w).Wrap)
		return m
	}
}

// ErrorRate generates an option that calculates the error rate percentile within
// a rolling time window to the load shedding calculation. If the error rate
// value falls between the lower and upper then a percentage of new requests
// will be rejected. The rolling window is configured by defining a bucket size
// and number of buckets. The preallocHint is an optimisation for keeping the
// number of alloc calls low. If the hint is zero then a default value is
// used.
func ErrorRate(lower float64, upper float64, bucketSize time.Duration, buckets int, preallocHint int, requiredPoints int) Option {
	return func(m *Loadshed) *Loadshed {
		if preallocHint < 1 {
			preallocHint = defaultHint
		}
		var errWindow = rolling.NewTimeWindow(bucketSize, buckets, preallocHint) // track err req in past time duration window
		var reqWindow = rolling.NewTimeWindow(bucketSize, buckets, preallocHint) // track req count in past time duration window

		var w = newErrRate(errWindow, reqWindow, requiredPoints, "ErrorRate", preallocHint)
		var a = rolling.NewPercentageRollup(w, lower, upper, "ChanceErrorRate")
		m.aggregators = append(m.aggregators, a)
		m.chain = append(m.chain, newErrorRateDecorator(errWindow, reqWindow).Wrap)
		return m
	}
}

// Concurrency generates an option that adds total concurrent requests to the
// load shedding calculation. Once the requests in flight reaches a value
// between lower and upper the Decorator will begin rejecting new requests
// based on the distance between the threshold values.
func Concurrency(lower int, upper int, wg *WaitGroup) Option {
	return func(m *Loadshed) *Loadshed {
		if wg == nil {
			wg = NewWaitGroup()
		}
		m.aggregators = append(m.aggregators, rolling.NewPercentageRollup(wg, float64(lower), float64(upper), "ChanceConcurrency"))
		m.chain = append(m.chain, newConcurrencyTrackingDecorator(wg).Wrap)
		return m
	}
}

// CPU generates an option that adds a rolling average of CPU usage to the
// load shedding calculation. It will configure the Decorator to reject a
// percentage of traffic once the average CPU usage is between lower and upper.
func CPU(lower float64, upper float64, pollingInterval time.Duration, windowSize int) Option {
	return func(m *Loadshed) *Loadshed {
		m.aggregators = append(m.aggregators, rolling.NewPercentageRollup(newAvgCPU(pollingInterval, windowSize), lower, upper, "ChanceCPU"))
		return m
	}
}

// Aggregator adds an arbitrary Aggregator to the evaluation for load shedding.
// The result of the aggregator will be interpreted as a percentage value
// between 0.0 and 1.0. This value will be used as the percentage of requests
// to reject.
func Aggregator(a rolling.Aggregator) Option {
	return func(m *Loadshed) *Loadshed {
		m.aggregators = append(m.aggregators, a)
		return m
	}
}

var zeroAggregator = rolling.NewSumRollup(rolling.NewPointWindow(1), "Zero")

// Loadshed is a struct containing all the aggregators that rejects a percentage of requests
// based on aggregation of system load data.
type Loadshed struct {
	random      func() float64
	aggregators []rolling.Aggregator
	chain       []func(func() error) func() error
}

// Do function inputs a function which returns an error
func (l *Loadshed) Do(runfn func() error) error {
	var result *rolling.Aggregate
	for _, aggregator := range l.aggregators {
		var r = aggregator.Aggregate()
		if result == nil || r.Value > result.Value {
			result = r
		}
	}
	var chance = l.random()
	if chance < result.Value {
		return Rejected{Aggregate: result}
	}
	for _, c := range l.chain {
		runfn = c(runfn)
	}
	return runfn()
}

// New generators a Loadshed struct that sheds load based on some
// definition of system load
func New(options ...Option) *Loadshed {
	var r = rand.New(rand.NewSource(time.Now().UnixNano()))
	var lo = &Loadshed{random: r.Float64}
	for _, option := range options {
		lo = option(lo)
	}

	if len(lo.aggregators) < 1 {
		lo.aggregators = append(lo.aggregators, zeroAggregator)
	}
	return lo

}
