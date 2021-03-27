package todo

import (
	"sync"
)

// Todo is a todo list
type Todo struct {
	newMsg bool
	lock   sync.Mutex
	wait   chan interface{}
}

// New Todo
func New() *Todo {
	return &Todo{
		newMsg: false,
		lock:   sync.Mutex{},
		wait:   make(chan interface{}, 1),
	}
}

// Ping something happened in your todo list
func (t *Todo) Ping() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.newMsg {
		return false
	}
	t.wait <- new(interface{})
	t.newMsg = true
	return true
}

// Done tell that you finished your todo list
func (t *Todo) Done() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.newMsg {
		return
	}
	t.newMsg = false
	if len(t.wait) > 0 { // flushing the chan
		<-t.wait
	}
}

// Wait for the next ping
func (t *Todo) Wait() chan interface{} {
	return t.wait
}
