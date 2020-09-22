package task

import (
	"context"
	"fmt"
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

// Add adds new task to list of tasks
func (ts *Tasks) Add(t Task) {
	ts.Lock()
	defer ts.Unlock()

	ts.items = append(ts.items, t)
}

// Kill cancel and supress a tasks in the list
func (ts *Tasks) Kill(i int) error {
	ts.Lock()
	defer ts.Unlock()

	// TODO: leaved here as a reminder
	// ts.items[i].Cancel()

	// reslice to remove item from list
	ts.items = append(ts.items[:i], ts.items[i+1:]...)

	return nil
}

// Filter and return tasks matching list of owners passed as parameters
func (ts *Tasks) Filter(owners ...string) []Task {
	ts.Lock()
	defer ts.Unlock()
	var t = []Task{}

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
func NewTask(o string) Task {
	return Task{
		Owner: o,
		Action: func(ctx context.Context) error {
			fmt.Println("action")
			return nil
		},
		// TODO: get this from request
		MaxExectionTime: 10,
		CPU:             1,
		RAM:             1,
	}
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
