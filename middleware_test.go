package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/atlassian/rolling"
)

func TestCPUOption(t *testing.T) {
	var o = CPU(50, 80, time.Second, 10)
	var m = &Middleware{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("cpu option did not add aggregate")
	}
}

func TestConcurrencyOption(t *testing.T) {
	var o = Concurrency(5000, 10000, nil)
	var m = &Middleware{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("concurrency option did not add aggregate")
	}
	if _, ok := m.next.(*concurrencyMiddleware); !ok {
		t.Fatalf("concurrency option did not install tracking middleware: %t", m.next)
	}
}

func TestAverageLatencyOption(t *testing.T) {
	var o = AverageLatency(.1, 1.0, time.Millisecond, 1000, 0)
	var m = &Middleware{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("avg latency option did not add aggregate")
	}
	if _, ok := m.next.(*latencyMiddleware); !ok {
		t.Fatalf("avg latency option did not install tracking middleware: %t", m.next)
	}
}

func TestPercentileLatencyOption(t *testing.T) {
	var o = PercentileLatency(.1, 1.0, time.Millisecond, 1000, 0, 99.9)
	var m = &Middleware{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("percentile latency option did not add aggregate")
	}
	if _, ok := m.next.(*latencyMiddleware); !ok {
		t.Fatalf("percentile latency option did not install tracking middleware: %t", m.next)
	}
}

func TestCallbackOption(t *testing.T) {
	var calledBack bool
	var c = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledBack = true
	})
	var o = Callback(c)
	var m = &Middleware{}
	m = o(m)
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	m.callback.ServeHTTP(w, r)
	if !calledBack {
		t.Fatalf("callback option did not set callback: %t", m.callback)
	}
}

func TestAggregatorOption(t *testing.T) {
	var w = rolling.NewPointWindow(1)
	var a = rolling.NewSumAggregator(w)
	var o = Aggregator(a)
	var m = &Middleware{}
	m = o(m)
	if len(m.aggregators) != 1 {
		t.Fatal("aggregator option did not add aggregate")
	}
	if m.aggregators[0] != a {
		t.Fatalf("aggregator options installed wrong aggregate: %t", m.aggregators[0])
	}
}

func TestMiddlwareDefaultOptions(t *testing.T) {
	var captured *Middleware
	var captureFN = func(m *Middleware) *Middleware {
		captured = m
		return m
	}
	var m = NewMiddlware(captureFN)
	var handler = m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if captured == nil {
		t.Fatal("did not capture middleware for test")
	}
	if len(captured.aggregators) != 1 {
		t.Fatal("default middleware did not add the zero aggregate")
	}
	if captured.aggregators[0] != zeroAggregator {
		t.Fatalf("default middleware did not add the zero aggregate: %t", captured.aggregators[0])
	}
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestMiddlewareDefaultCallback(t *testing.T) {
	var window = rolling.NewPointWindow(1)
	window.Feed(1)
	var m = NewMiddlware(Aggregator(rolling.NewSumAggregator(window)))
	var handler = m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var r, _ = http.NewRequest(http.MethodGet, "/", nil)
	var w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("middleware did not call default callback: %d", w.Code)
	}
}
