package task

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// DummyAction is the most basic action, used for tests and illustration purpose
type DummyAction struct {
	Name     string  `json:"name"`
	Wait     float64 `json:"wait"`
	Counter  int64   `json:"counter"`
	ExitCode int     `json:"exit_code"`
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

func (r *DummyRun) Wait(ctx context.Context) (Status, error) {
	waiter := make(chan interface{})
	r.da.waiters = append(r.da.waiters, waiter)
	var status Status
	select {
	case <-waiter:
		fmt.Println("done")
		status = Done
	case <-ctx.Done():
		switch ctx.Err() {
		case context.Canceled:
			fmt.Println("canceled")
			status = Canceled
		case context.DeadlineExceeded:
			fmt.Println("timeout")
			status = Timeout
		}
	}

	return status, nil
}

func (r DummyRun) Down() error {
	return nil
}

// Run action interface implementation
func (da *DummyAction) Up(pwd string, environments map[string]string) (Run, error) {
	// Print name
	fmt.Println("DummyAction :", da.Name)
	if da.waiters == nil {
		da.waiters = make([]chan interface{}, 0)
	}
	if da.Wait == 0 {
		da.Wait = 0.1
	}
	go func() {
		// Sleep
		time.Sleep(time.Duration(da.Wait) * time.Millisecond)
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
