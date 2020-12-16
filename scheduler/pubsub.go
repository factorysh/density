package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Event struct {
	Action string    `json:"action"`
	Id     uuid.UUID `json:"id"`
}

type PubSub struct {
	lock        *sync.Mutex
	cpt         uint64
	subscribers map[uint64]chan Event
	wg          *sync.WaitGroup
}

func NewPubSub() *PubSub {
	return &PubSub{
		lock:        &sync.Mutex{},
		cpt:         0,
		subscribers: make(map[uint64]chan Event),
		wg:          &sync.WaitGroup{},
	}
}

func (p *PubSub) Subscribe(ctx context.Context) chan Event {
	p.lock.Lock()
	id := p.cpt
	p.cpt++
	p.subscribers[id] = make(chan Event)
	p.wg.Add(1)
	p.lock.Unlock()
	go func() {
		<-ctx.Done()
		p.lock.Lock()
		delete(p.subscribers, id)
		p.wg.Done()
		p.lock.Unlock()
	}()
	return p.subscribers[id]
}

func (p *PubSub) Publish(evt Event) {
	fmt.Println("publish", evt)
	p.lock.Lock()
	defer p.lock.Unlock()
	// Warning, chans are blocking
	for _, c := range p.subscribers {
		c <- evt
	}
}

func (p *PubSub) Wait() {
	p.wg.Wait()
}
