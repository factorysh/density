package task

import (
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"
)

// DummyAction is the most basic action, used for tests and illustration purpose
type DummyAction struct {
	Name        string `json:"name"`
	Wait        int    `json:"wait"`
	Counter     int64  `json:"counter"`
	WithTimeout bool   `json:"with_timeout"`
	Status      string `json:"status"`
	WithCommand bool   `json:"with_command"`
	ExitCode    int    `json:"exit_code"`
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
		cmd := exec.CommandContext(ctx, "sleep", "2")
		_ = cmd.Run()
		da.ExitCode = cmd.ProcessState.ExitCode()

	}

	return nil
}
