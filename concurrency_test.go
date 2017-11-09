package loadshed

import "testing"

func TestWaitGroup(t *testing.T) {
	var wg = NewWaitGroup()
	wg.Add(1)
	if wg.Aggregate().Value != 1 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate().Value)
	}
	wg.Done()
	if wg.Aggregate().Value != 0 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate().Value)
	}
}

func TestConcurrencyDecorator(t *testing.T) {
	var wg = NewWaitGroup()
	var decorator = newConcurrencyTrackingDecorator(wg)

	var d = decorator.Wrap(func() error {
		if wg.Aggregate().Value != 1 {
			t.Fatalf("wrong internal count: %f", wg.Aggregate().Value)
		}
		return nil
	})
	d()
	if wg.Aggregate().Value != 0 {
		t.Fatalf("wrong internal count: %f", wg.Aggregate().Value)
	}
}
