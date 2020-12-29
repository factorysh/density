package task

import (
	"context"
)

// Action interface describe behavior of a job
type Action interface {
	// Validate if attributes are correct
	Validate() error
	// Run with a context, a working directory and environments variables.
	Up(pwd string, environments map[string]string) (Run, error)
}

type Run interface {
	Down() error
	Wait(context.Context) error
}
