//go:generate stringer -type=Status
package task

type Status int

const (
	Waiting  Status = 1
	Running  Status = 2
	Done     Status = 3
	Canceled Status = 4
	Error    Status = 5
)
