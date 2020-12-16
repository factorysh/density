package task

import (
	"context"
)

// Action interface describe behavior of a job
type Action interface {
	Validate() error
	Run(ctx context.Context, pwd string, environments map[string]string) error
}
