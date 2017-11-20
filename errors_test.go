package loadshed

import "testing"

func TestRejectedError(t *testing.T) {
	var e = &Rejected{Aggregate: zeroAggregator.Aggregate()}
	if e.Error() != "request rejected Zero is 0.000000" {
		t.Fatalf("Got unexpected error %s", e.Error())
	}
}
