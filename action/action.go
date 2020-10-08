package action

import (
	"context"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/pkg/errors"
)

// Actions implement task.Action interface

// Kind represents all kinds of supported actions
type Kind int

const (
	// DockerCompose kind
	DockerCompose Kind = iota
)

type contextKey string

var (
	// ContextUUID is used to put a uuid value in a context
	ContextUUID = contextKey("uuid")
)

// FromCtxUUID fetch an uuid from a context value
func FromCtxUUID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextUUID).(string)
	return v, ok
}

// NewAction creates a specific job from a job description
func NewAction(k Kind, desc []byte) (task.Action, error) {

	switch k {
	case DockerCompose:
		compose, err := NewCompose(desc)
		if err != nil {
			return nil, err
		}
		return compose, err

	default:
		return nil, errors.New("This kind of job is not supported")
	}
}
