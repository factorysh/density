package task

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tasks is a list of tasks
type Tasks struct {
	sync.RWMutex
	items []Task
}

// NewTasks inits a list of tasks
func NewTasks() *Tasks {
	return &Tasks{
		items: []Task{},
	}
}

// List all tasks in this pool
func (ts *Tasks) List() []Task {
	ts.Lock()
	defer ts.Unlock()

	return ts.items

}

// Filter and return tasks matching list of owners passed as parameters
func (ts *Tasks) Filter(owners ...string) (t []Task) {
	ts.Lock()
	defer ts.Unlock()

	for _, task := range ts.items {
		for _, owner := range owners {
			if task.Owner == owner {
				t = append(t, task)
			}
		}
	}

	return t
}

// Task something to do
type Task struct {
	Start           time.Time          // Start time
	MaxWaitTime     time.Duration      // Max wait time before starting Action
	MaxExectionTime time.Duration      // Max execution time
	CPU             int                // CPU quota
	RAM             int                // RAM quota
	Action          Action             // Action is an abstract, the thing to do
	Id              uuid.UUID          // Id
	Cancel          context.CancelFunc // Cancel the action
	Status          Status             // Status
	Mtime           time.Time          // Modified time
	Owner           string             // Owner
	resourceCancel  context.CancelFunc
}

// Action does something
type Action func(context.Context) error

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
