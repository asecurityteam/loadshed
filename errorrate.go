package loadshed

import (
	"fmt"

	"bitbucket.org/atlassian/rolling"
)

// errRate is a struct representing the different feeders and aggregators to calculate error rate
type errRate struct {
	name      string
	errRollup rolling.Aggregator
	reqRollup rolling.Aggregator
}

// Aggregate calculates the error rate
func (era *errRate) Aggregate() *rolling.Aggregate {

	var req = era.reqRollup.Aggregate()
	var err = era.errRollup.Aggregate()
	var errRate = 0.0
	if req.Value != 0 {
		errRate = (err.Value / req.Value) * 100
	}

	err.Source = req
	return &rolling.Aggregate{
		Source: err,
		Name:   era.name,
		Value:  errRate,
	}
}

// Name returns the name of the aggregator
func (era *errRate) Name() string {
	return era.name
}

// newErrRate creates a new aggregator to calculate error rate
// errWindow - feeder for errors
// reqWindow - feeder for requests
// minReqCount - minimum number of requests required in a time window
// name - name of the aggregator
// preallocHint - preallocation hint
func newErrRate(errWindow rolling.Window, reqWindow rolling.Window, minReqCount int, name string, preallocHint int) *errRate {
	var errRollup = rolling.NewLimitedRollup(minReqCount, reqWindow, rolling.NewSumRollup(errWindow, fmt.Sprintf("%s-error-count", name)))
	var reqRollup = rolling.NewSumRollup(reqWindow, fmt.Sprintf("%s-req-count", name))

	return &errRate{
		errRollup: errRollup,
		reqRollup: reqRollup,
		name:      name,
	}
}

type errorRateDecorator struct {
	errFeeder rolling.Feeder
	reqFeeder rolling.Feeder
}

func (h *errorRateDecorator) Wrap(next func() error) func() error {
	return func() error {
		h.reqFeeder.Feed(1)
		var e error
		e = next()
		if e != nil {
			h.errFeeder.Feed(1)
		}
		return e
	}
}

// newErrorRateDecorator tracks error rates of an action using a given two
// rolling window.Feeder.
func newErrorRateDecorator(errFeeder rolling.Feeder, reqFeeder rolling.Feeder) wrapper {
	return &errorRateDecorator{reqFeeder: reqFeeder, errFeeder: errFeeder}
}
