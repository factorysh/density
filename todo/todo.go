package todo

import "sync"

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
	}
	t.newMsg = true
}

func (t *Todo) Done() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.newMsg = false
}

func (t *Todo) Wait() chan interface{} {
	return t.wait
}
