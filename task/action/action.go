package action

import (
	"context"

	"github.com/factorysh/density/task/run"
	"github.com/factorysh/density/task/status"
)

// Action interface describe behavior of a job
type Action interface {
	// Validate if attributes are correct
	Validate() error
	// Run with a context, a working directory and environments variables.
	Up(pwd string, environments map[string]string, runID int) (run.Run, error)
	// RegisteredName is registered name
	RegisteredName() string
}

type Run interface {
	Down() error
	Wait(context.Context) (status.Status, error)
}
