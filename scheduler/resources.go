package scheduler

import (
	"context"
	"errors"
	"sync"
)

type Resources struct {
	TotalRAM  int
	ram       int
	TotalCPU  int
	cpu       int
	processes int
	lock      *sync.RWMutex
}

func NewResources(cpu, ram int) *Resources {
	return &Resources{
		TotalRAM:  ram,
		ram:       ram,
		TotalCPU:  cpu,
		cpu:       cpu,
		processes: 0,
		lock:      &sync.RWMutex{},
	}
}

func (r *Resources) Check(cpu, ram int) error {
	if cpu <= 0 {
		return errors.New("CPU must be > 0")
	}
	if cpu > r.TotalCPU {
		return errors.New("Too much CPU is required")
	}
	if ram <= 0 {
		return errors.New("RAM must be > 0")
	}
	if ram > r.TotalRAM {
		return errors.New("Too much RAM is required")
	}
	return nil
}

func (r *Resources) Consume(ctx context.Context, cpu, ram int) {
	r.lock.Lock()
	r.cpu -= cpu
	r.ram -= ram
	r.processes++
	r.lock.Unlock()
	go func() {
		select {
		case <-ctx.Done():
			r.lock.Lock()
			r.cpu += cpu
			r.ram += ram
			r.processes--
			r.lock.Unlock()
		}
	}()
}

func (r *Resources) IsDoable(cpu, ram int) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return cpu <= r.cpu && ram <= r.ram
}
