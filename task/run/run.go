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
	ExitCode int       `json:"exit_code"`
	Runner   string    `json:"runner"`
}

type Run interface {
	Down() error
	Wait(context.Context) (status.Status, error)
	ID() (string, error)
	RegisteredName() string
	Status() (Status, int, error)
	Data() Data
}
