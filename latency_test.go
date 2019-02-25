package loadshed

import (
	"fmt"
	"testing"
	"time"

	"github.com/asecurityteam/rolling"
)

func TestLatencyDecorator(t *testing.T) {
	var window = rolling.NewPointWindow(1)
	var decorator = newLatencyTrackingDecorator(window)
	var wrap = decorator.Wrap(func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	var e = wrap()
	if e != nil {
		t.Fatal("Unexpected error")
	}
	var a = rolling.NewSumRollup(window, "")
	var result = a.Aggregate().Value
	if result < (5*time.Millisecond).Seconds() || result > (6*time.Millisecond).Seconds() {
		t.Fatalf("incorrect latency record: %f", result)
	}
}

func TestLatencyDecoratorError(t *testing.T) {
	var window = rolling.NewPointWindow(1)
	var decorator = newLatencyTrackingDecorator(window)
	var wrap = decorator.Wrap(func() error {
		time.Sleep(5 * time.Millisecond)
		return fmt.Errorf("")
	})
	var e = wrap()
	if e == nil {
		t.Fatal("Expected error")
	}
	var a = rolling.NewSumRollup(window, "")
	var result = a.Aggregate().Value
	if result < (5*time.Millisecond).Seconds() || result > (6*time.Millisecond).Seconds() {
		t.Fatalf("incorrect latency record: %f", result)
	}
}
