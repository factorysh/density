package run

import (
	"context"
	"time"

	"github.com/factorysh/density/task/status"
)

// Data is struct used to specified required run data that abstraction should provide
type Data struct {
	Start    time.Time `json:"start"`
	Finish   time.Time `json:"finish"`
	ID       int       `json:"id"`
	ExitCode int       `json:"exit_code"`
	Runner   string    `json:"runner"`
	Running  bool      `json:"running"`
}

type Run interface {
	Down() error
	Wait(context.Context) (status.Status, error)
	RunnerID() (string, error)
	RegisteredName() string
	Status() (Status, int, error)
	Data() Data
}
