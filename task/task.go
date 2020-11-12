package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Task something to do
type Task struct {
	Start           time.Time          // Start time
	MaxWaitTime     time.Duration      // Max wait time before starting Action
	MaxExectionTime time.Duration      // Max execution time
	CPU             int                // CPU quota
	RAM             int                // RAM quota
	Action          Action             `json:"-"` // Action is an abstract, the thing to do
	Id              uuid.UUID          // Id
	Cancel          context.CancelFunc `json:"-"` // Cancel the action
	Status          Status             // Status
	Mtime           time.Time          // Modified time
	Owner           string             // Owner
	Retry           int                // Number of retry before crash
	Every           time.Duration      // Periodic execution. Exclusive with Cron
	Cron            string             // Cron definition. Exclusive with Every
	resourceCancel  context.CancelFunc
}

// NewTask init a new task
func NewTask(o string, a Action) Task {
	t, _ := New()
	t.Owner = o
	t.Action = a
	return *t
}

func New() (*Task, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	return &Task{
		CPU:    1,
		RAM:    1,
		Status: Waiting,
		Mtime:  time.Now(),
		Id:     id,
	}, nil
}

type TaskByStart []*Task

func (t TaskByStart) Len() int           { return len(t) }
func (t TaskByStart) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TaskByStart) Less(i, j int) bool { return t[i].Start.Before(t[j].Start) }

type TaskByKarma []*Task

func (t TaskByKarma) Len() int      { return len(t) }
func (t TaskByKarma) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t TaskByKarma) Less(i, j int) bool {
	return (t[i].RAM * t[i].CPU / int(int64(t[i].MaxExectionTime))) <
		(t[j].RAM * t[j].CPU / int(int64(t[j].MaxExectionTime)))
}
