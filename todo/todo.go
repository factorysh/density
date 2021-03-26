package todo

import (
	"errors"
	"sync"
)

type Todo struct {
	newMsg bool
	lock   sync.Mutex
	wait   chan interface{}
}

func New() *Todo {
	return &Todo{
		newMsg: false,
		lock:   sync.Mutex{},
		wait:   make(chan interface{}, 1),
	}
}

func (t *Todo) Ping() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.newMsg {
		t.wait <- new(interface{})
		t.newMsg = true
	}
}

func (t *Todo) Done() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.newMsg = false
	if !t.newMsg {
		return errors.New("double release")
	}
	if len(t.wait) > 0 { // flushing the chan
		<-t.wait
	}
	return nil
}

func (t *Todo) Wait() chan interface{} {
	return t.wait
}
