package task

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
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
	return Task{
		Owner:  o,
		Action: a,
		// TODO: get this from request
		MaxExectionTime: 10,
		CPU:             1,
		RAM:             1,
	}
}

// Action interface describe behavior of a job
type Action interface {
	Validate() (string, error)
	Run(ctx context.Context) error
}

// DummyAction is the most basic action, used for tests and illustration purpose
type DummyAction struct {
	Name        string
	Wait        int
	Wg          *sync.WaitGroup
	Counter     int64
	WithTimeout bool
	Status      string
	WithCommand bool
	ExitCode    int
}

// Validate action interface implementation
func (da *DummyAction) Validate() (string, error) {
	return "", nil
}

// Run action interface implementation
func (da *DummyAction) Run(ctx context.Context) error {
	// Print name
	fmt.Println(da.Name)
	// Sleep
	time.Sleep(time.Duration(da.Wait) * time.Millisecond)

	// Add to dedicated counter
	atomic.AddInt64(&da.Counter, 1)

	// Handle timeout if specified needed
	if da.WithTimeout {
		select {
		case <-time.After(2 * time.Second):
			fmt.Println("2s")
			da.Status = "waiting"
		case <-ctx.Done():
			fmt.Println("canceled")
			da.Status = "canceled"
		}
	}

	if da.WithCommand {
		cmd := exec.CommandContext(ctx, "sleep", "5")
		_ = cmd.Run()
		da.ExitCode = cmd.ProcessState.ExitCode()

	}

	// Tell to WG that work is done
	if da.Wg != nil {
		da.Wg.Done()
	}

	return nil
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
