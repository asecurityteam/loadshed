package loadshed

import (
	"fmt"

	"bitbucket.org/atlassian/rolling"
)

// Rejected is error returned when a request is rejected because of load shedding
type Rejected struct {
	Aggregate *rolling.Aggregate
}

func (r Rejected) Error() string {
	var aggregate = r.Aggregate
	var reason = fmt.Sprintf("request rejected %s is %f", aggregate.Name, aggregate.Value)
	for aggregate.Source != nil {
		aggregate = aggregate.Source
		reason = fmt.Sprintf("%s because %s is %f", reason, aggregate.Name, aggregate.Value)
	}
	return reason
}
