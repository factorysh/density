//go:generate stringer -type=Status
package task

import (
	"encoding/json"
	"fmt"
)

type Status int

const (
	Waiting  Status = 0
	Running  Status = 1
	Done     Status = 2
	Timeout  Status = 3
	Canceled Status = 4
	Error    Status = 5
)

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var raw string
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	var start uint8
	for i, end := range _Status_index[1:] {
		m := _Status_name[start:end]
		fmt.Println(raw, m, i)
		if m == raw {
			*s = Status(i)
			fmt.Println("debug:", i, s.String())
			return nil
		}
		start = end
	}
	return fmt.Errorf("Not a known status: %s", raw)
}
