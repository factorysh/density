package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Task something to do
type Task struct {
	Start           time.Time
	MaxWaitTime     time.Duration
	MaxExectionTime time.Duration
	CPU             int
	RAM             int
	Action          Action
	Id              uuid.UUID
}

// Action does something
type Action func(context.Context) error
