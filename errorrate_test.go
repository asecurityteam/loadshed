package loadshed

import (
	"fmt"
	"testing"
	"time"

	"bitbucket.org/atlassian/rolling"
)

func TestErrorRateNoError(t *testing.T) {
	var bucketSize = time.Millisecond
	var timeWindow = 5
	var preallocHint = 5

	var errWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var reqWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var decorator = newErrorRateDecorator(errWindow, reqWindow)
	var wrap = decorator.Wrap(func() error {
		return nil
	})
	var e = wrap()
	if e != nil {
		t.Fatal("Unexpected error")
	}
	var a = rolling.NewSumRollup(errWindow, "")
	var eresult = a.Aggregate().Value
	if int(eresult) != 0 {
		t.Fatalf("Unexpected result %f", eresult)
	}
	var b = rolling.NewSumRollup(reqWindow, "")
	var result = b.Aggregate().Value
	if int(result) != 1 {
		t.Fatalf("Unexpected result %f", eresult)
	}
}

func TestErrorRateError(t *testing.T) {
	var bucketSize = time.Millisecond
	var timeWindow = 5
	var preallocHint = 5

	var errWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var reqWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var decorator = newErrorRateDecorator(errWindow, reqWindow)
	var wrap = decorator.Wrap(func() error {
		return fmt.Errorf("")
	})
	var e = wrap()
	if e == nil {
		t.Fatal("Expected error")
	}
	var a = rolling.NewSumRollup(errWindow, "")
	var eresult = a.Aggregate().Value
	if int(eresult) != 1 {
		t.Fatalf("Unexpected result %f", eresult)
	}
	var b = rolling.NewSumRollup(reqWindow, "")
	var result = b.Aggregate().Value
	if int(result) != 1 {
		t.Fatalf("Unexpected result %f", eresult)
	}
}

func TestErrorRateErrorTimeBucket(t *testing.T) {
	var bucketSize = time.Millisecond
	var timeWindow = 5
	var preallocHint = 5

	var errWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var reqWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var decorator = newErrorRateDecorator(errWindow, reqWindow)
	var wrap = decorator.Wrap(func() error {
		time.Sleep(time.Duration(timeWindow+1) * bucketSize)
		return fmt.Errorf("")
	})
	var e = wrap()
	if e == nil {
		t.Fatal("Expected error")
	}
	var a = rolling.NewSumRollup(errWindow, "")
	var eresult = a.Aggregate().Value
	if int(eresult) != 1 {
		t.Fatalf("Unexpected result %f", eresult)
	}
	var b = rolling.NewSumRollup(reqWindow, "")
	var result = b.Aggregate().Value
	if int(result) != 1 {
		t.Fatalf("Unexpected result %f", eresult)
	}
}

func TestErrRateRollUp(t *testing.T) {
	var minReqCount = 2
	var bucketSize = time.Millisecond
	var timeWindow = 5
	var name = "test"
	var preallocHint = 5

	var errWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var reqWindow = rolling.NewTimeWindow(bucketSize, timeWindow, preallocHint)
	var rollup = newErrRate(errWindow, reqWindow, minReqCount, name, preallocHint)
	errWindow.Feed(1)
	reqWindow.Feed(1)
	if rollup.Aggregate().Value != 0 {
		t.Fatalf("Minimum request count not met. Aggregate value: %f", rollup.Aggregate().Value)
	}
	errWindow.Feed(1)
	reqWindow.Feed(1)
	if rollup.Aggregate().Value != 100 {
		t.Fatalf("All requests errored. Aggregate value: %f", rollup.Aggregate().Value)
	}
	reqWindow.Feed(1)
	reqWindow.Feed(1)

	if rollup.Aggregate().Value != 50 {
		t.Fatalf("Half requests errored. Aggregate value: %f", rollup.Aggregate().Value)
	}
	time.Sleep(6 * time.Millisecond) //roll window
	if rollup.Aggregate().Value != 0 {
		t.Fatalf("Roll window. Aggregate value: %f", rollup.Aggregate().Value)
	}

	if rollup.Name() != name {
		t.Fatalf("Expected name %s but got %s", name, rollup.Name())
	}
}
