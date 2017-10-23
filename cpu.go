package loadshed

import (
	"time"

	"bitbucket.org/atlassian/rolling"
	pscpu "github.com/shirou/gopsutil/cpu"
)

// AvgCPU is a rolling average Aggregator for AvgCPU usage of a host.
type AvgCPU struct {
	pollingInterval time.Duration
	feeder          rolling.Feeder
	aggregator      rolling.Aggregator
}

func (c *AvgCPU) poll() {
	for {
		c.feed()
	}
}

func (c *AvgCPU) feed() {
	var pctUsed, _ = pscpu.Percent(c.pollingInterval, false)
	c.feeder.Feed(pctUsed[0])
}

// Aggregate emits the current rolling average of AvgCPU usage
func (c *AvgCPU) Aggregate() float64 {
	return c.aggregator.Aggregate()
}

// NewAvgCPU tracks a rolling average of AvgCPU consumption. The time window is
// defined as windowSize * pollingInterval.
func NewAvgCPU(pollingInterval time.Duration, windowSize int) *AvgCPU {
	var w = rolling.NewPointWindow(windowSize)
	var a = rolling.NewAverageAggregator(w)
	var result = &AvgCPU{pollingInterval, w, a}
	go result.poll()
	return result
}
