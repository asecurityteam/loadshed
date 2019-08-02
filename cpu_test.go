package loadshed

import (
	"math"
	"testing"
	"time"

	"github.com/asecurityteam/rolling"
)

func TestCPU(t *testing.T) {
	// t.Skip("test is too flaky to run. ticket in the backlog")
	var points = 3
	var w = rolling.NewPointWindow(points)
	var a = rolling.NewAverageRollup(w, "")
	var c = &avgCPU{pollingInterval: time.Second, feeder: w, rollup: a}

	for x := 0; x < points+1; x = x + 1 {
		c.feed()
	}
	var result = c.Aggregate().Value
	if result <= 0 || result > 100 {
		t.Fatalf("invalid AvgCPU percentage: %f", result)
	}
}

func TestCPUPolling(t *testing.T) {
	// t.Skip("test is too flaky to run. ticket in the backlog")
	var c = newAvgCPU(time.Millisecond, 5)
	c.feed()
	var baseline = c.Aggregate().Value
	var stop = make(chan bool)
	go func(stop chan bool) {
		for {
			select {
			case <-stop:
				return
			default:
				// Run some CPU bound operations to generate data
				for x := 0; x < 100; x = x + 1 {
					math.Pow(10, 1000)
				}
			}
		}
	}(stop)
	time.Sleep(3 * time.Second)
	var result = c.Aggregate().Value
	close(stop)
	if result <= baseline {
		t.Fatalf("AvgCPU never increased: %f - %f", baseline, result)
	}
}
