package action

import (
	"github.com/factorysh/batch-scheduler/task"
	"github.com/pkg/errors"
)

// Actions implement task.Action interface

// Description wraps all fields a used to parse a job description
type Description struct {
	// DockerCompose field for a docker-compose yaml file as string
	DockerCompose string `json:"docker-compose"`
}

// NewAction creates a specific job from a job description
func NewAction(desc Description) (task.Action, error) {

	if desc.DockerCompose != "" {
		compose, err := NewCompose(desc)
		if err != nil {
			return nil, err
		}

		return compose, err
	}

	return nil, errors.New("This kind of job is not supported")
}
