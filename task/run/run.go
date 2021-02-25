package run

import (
	"context"

	"github.com/factorysh/density/task/status"
)

type Run interface {
	Down() error
	Wait(context.Context) (status.Status, error)
	ID() (string, error)
	RegisteredName() string
	Status() (Status, int, error)
}
