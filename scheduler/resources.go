package scheduler

import "sync"

type Resources struct {
	TotalRAM int
	TotalCPU int
	lock     sync.RWMutex
}
