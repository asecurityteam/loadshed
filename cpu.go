package loadshed

import (
	"time"

	"bitbucket.org/atlassian/rolling"
	pscpu "github.com/shirou/gopsutil/cpu"
)

// avgCPU is a rolling average Aggregator for avgCPU usage of a host.
type avgCPU struct {
	pollingInterval time.Duration
	feeder          rolling.Feeder
	rollup          rolling.Rollup
}

func (c *avgCPU) poll() {
	for {
		c.feed()
	}
}

func (c *avgCPU) feed() {
	var pctUsed, _ = pscpu.Percent(c.pollingInterval, false)
	c.feeder.Feed(pctUsed[0])
}

// Name emits the rollup name for identification.
func (c *avgCPU) Name() string {
	return c.rollup.Name()
}

// Aggregate emits the current rolling average of avgCPU usage
func (c *avgCPU) Aggregate() *rolling.Aggregate {
	return c.rollup.Aggregate()
}

// newavgCPU tracks a rolling average of avgCPU consumption. The time window is
// defined as windowSize * pollingInterval.
func newAvgCPU(pollingInterval time.Duration, windowSize int) *avgCPU {
	var w = rolling.NewPointWindow(windowSize)
	var a = rolling.NewAverageRollup(w, "AverageCPU")
	var result = &avgCPU{pollingInterval, w, a}
	go result.poll()
	return result
}
