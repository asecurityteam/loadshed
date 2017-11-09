package loadshed

import (
	"sync/atomic"
	"testing"
	"time"

	"bitbucket.org/atlassian/rolling"
)

func TestCPUOption(t *testing.T) {
	var o = CPU(50, 80, time.Second, 10)
	var l = &Loadshed{}
	l = o(l)
	if len(l.aggregators) != 1 {
		t.Fatal("cpu option did not add aggregate")

	}
}

func TestConcurrencyOption(t *testing.T) {
	var o = Concurrency(5000, 10000, nil)
	var m = &Loadshed{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("concurrency option did not add aggregate")
	}
	if len(m.chain) != 1 {
		t.Fatal("percentile latency option did not add chain")
	}
}

func TestAverageLatencyOption(t *testing.T) {
	var o = AverageLatency(.1, 1.0, time.Millisecond, 1000, 0, 0)
	var m = &Loadshed{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("avg latency option did not add aggregate")
	}
	if len(m.chain) != 1 {
		t.Fatal("percentile latency option did not add chain")
	}
}

func TestPercentileLatencyOption(t *testing.T) {
	var o = PercentileLatency(.1, 1.0, time.Millisecond, 1000, 0, 0, 99.9)
	var m = &Loadshed{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("percentile latency option did not add aggregate")
	}
	if len(m.chain) != 1 {
		t.Fatal("percentile latency option did not add chain")
	}
}

func TestLoadshedAggregator(t *testing.T) {
	var w = rolling.NewPointWindow(1)
	var a = rolling.NewSumRollup(w, "")
	var o = Aggregator(a)
	var m = &Loadshed{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("percentile latency option did not add aggregate")
	}
	if m.aggregators[0] != a {
		t.Fatalf("aggregator options installed wrong aggregate: %t", m.aggregators[0])
	}
}

func TestErrorRateOption(t *testing.T) {
	var o = ErrorRate(50, 75, time.Millisecond, 10, 10, 1)
	var m = &Loadshed{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("percentile latency option did not add aggregate")
	}
	if len(m.chain) != 1 {
		t.Fatal("percentile latency option did not add chain")
	}
}

func TestLoadshed(t *testing.T) {
	var option = &fakeOption{}
	var l = New(option.Option())
	if option.Counter != 1 {
		t.Fatal("Option not installed")
	}
	var e = l.Do(func() error { return nil })
	if e != nil {
		t.Fatalf("Unexpected error %s", e)
	}
}

func TestLoadshedRequestRejected(t *testing.T) {
	var option = &fakeOption{err: true}
	var l = New(option.Option())
	var e = l.Do(func() error { return nil })
	switch e.(type) {
	case Rejected:
		//pass
	default:
		t.Fatal("Did not get expected error")
	}
}

type fakeOption struct {
	Counter int32
	err     bool
}

func (f *fakeOption) Option() Option {
	return func(m *Loadshed) *Loadshed {
		var w = rolling.NewPointWindow(1)
		var zeroAggregator = rolling.NewSumRollup(w, "Zero")
		if f.err {
			w.Feed(1)
		}
		m.aggregators = append(m.aggregators, zeroAggregator)
		atomic.AddInt32(&f.Counter, 1)
		return m
	}
}
