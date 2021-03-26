package todo

import (
	"errors"
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
	if !t.newMsg {
		t.wait <- new(interface{})
		t.newMsg = true
		return true
	}
	return false
}

// Done tell that you finished your todo list
func (t *Todo) Done() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.newMsg {
		return errors.New("double release")
	}
	t.newMsg = false
	if len(t.wait) > 0 { // flushing the chan
		<-t.wait
	}
	return nil
}

// Wait for the next ping
func (t *Todo) Wait() chan interface{} {
	return t.wait
}
