package task

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/factorysh/density/task/action"
	"github.com/factorysh/density/task/run"
	_status "github.com/factorysh/density/task/status"
)

var _ action.Action = &DummyAction{}
var _ action.Run = &DummyRun{}

// DummyAction is the most basic action, used for tests and illustration purpose
type DummyAction struct {
	Name     string        `json:"name"`
	Wait     time.Duration `json:"wait"`
	Counter  int64         `json:"counter"`
	ExitCode int           `json:"exit_code"`
	waiters  []chan interface{}
}

func (da *DummyAction) RegisteredName() string {
	return "dummy"
}

// Validate action interface implementation
func (da *DummyAction) Validate() error {
	return nil
}

type DummyRun struct {
	da *DummyAction
}

func (r *DummyRun) Data() run.Data {
	return run.Data{}
}

func (r *DummyRun) ID() (string, error) {
	return fmt.Sprintf("Run of %s", r.da.Name), nil
}

func (r *DummyRun) Status() (run.Status, int, error) {
	return run.Running, 0, nil
}

func (r *DummyRun) RegisteredName() string {
	return "dummy"
}

func (r *DummyRun) Wait(ctx context.Context) (_status.Status, error) {
	waiter := make(chan interface{})
	r.da.waiters = append(r.da.waiters, waiter)
	var status _status.Status
	select {
	case <-waiter:
		fmt.Printf("DummyRun.Wait %s done\n", r.da.Name)
		status = _status.Done
	case <-ctx.Done():
		switch ctx.Err() {
		case context.Canceled:
			fmt.Printf("DummyRun.Wait %s canceled\n", r.da.Name)
			status = _status.Canceled
		case context.DeadlineExceeded:
			fmt.Printf("DummyRun.Wait %s timeout\n", r.da.Name)
			status = _status.Timeout
		}
	}
	return status, nil
}

func (r DummyRun) Down() error {
	return nil
}

// Run action interface implementation
func (da *DummyAction) Up(pwd string, environments map[string]string) (run.Run, error) {
	// Print name
	fmt.Println("DummyAction.Up :", da.Name)
	if da.waiters == nil {
		da.waiters = make([]chan interface{}, 0)
	}
	if da.Wait == 0 {
		da.Wait = 100 * time.Microsecond
	}
	go func() {
		// Sleep
		time.Sleep(da.Wait)
		// Add to dedicated counter
		atomic.AddInt64(&da.Counter, 1)
		for _, waiter := range da.waiters {
			waiter <- new(interface{})
		}
	}()

	return &DummyRun{
		da: da,
	}, nil
}
