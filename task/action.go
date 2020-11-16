package task

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Action interface describe behavior of a job
type Action interface {
	Validate() error
	Run(ctx context.Context, pwd string, environments map[string]string) error
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
func (da *DummyAction) Validate() error {
	return nil
}

// Run action interface implementation
func (da *DummyAction) Run(ctx context.Context, pwd string, environments map[string]string) error {
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
