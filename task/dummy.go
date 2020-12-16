package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

type rawWaiter struct {
	Cpt int `json:"cpt"`
}

type Waiter struct {
	wg   *sync.WaitGroup
	cpt  int
	lock *sync.Mutex
}

func NewWaiter() *Waiter {
	return &Waiter{
		wg:   &sync.WaitGroup{},
		cpt:  0,
		lock: &sync.Mutex{},
	}
}

func (w *Waiter) Add(delta int) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.cpt += delta
	w.wg.Add(delta)
}

func (w *Waiter) Done() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.wg.Done()
	w.cpt--
}

func (w *Waiter) Wait() {
	w.wg.Wait()
}

func (w *Waiter) UnmarshalJSON(b []byte) error {
	var r rawWaiter
	err := json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	w.wg = &sync.WaitGroup{}
	w.lock = &sync.Mutex{}
	w.Add(r.Cpt)
	return nil
}

func (w *Waiter) MarshalJSON() ([]byte, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	r := rawWaiter{Cpt: w.cpt}
	return json.Marshal(r)
}

// DummyAction is the most basic action, used for tests and illustration purpose
type DummyAction struct {
	Name        string  `json:"name"`
	Wait        int     `json:"wait"`
	Wg          *Waiter `json:"wg"`
	Counter     int64   `json:"counter"`
	WithTimeout bool    `json:"with_timeout"`
	Status      string  `json:"status"`
	WithCommand bool    `json:"with_command"`
	ExitCode    int     `json:"exit_code"`
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
