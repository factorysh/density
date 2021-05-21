package pubsub

import (
	"context"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
	go func(id uint64) {
		<-ctx.Done() // closing the subscription
		p.lock.Lock()
		delete(p.subscribers, id)
		p.wg.Done()
		p.lock.Unlock()
		log.WithField("id", id).Info("Closing subscribtion")
	}(id)
	log.WithField("id", id).WithField("subscribers", len(p.subscribers)).Info("Opening subscribtion")
	return p.subscribers[id]
}

func (p *PubSub) Publish(evt Event) {
	p.lock.Lock()
	log.WithField("event", evt).WithField("subscribers", len(p.subscribers)).Info("publish")
	defer p.lock.Unlock()
	// Warning, chans are blocking
	for _, c := range p.subscribers {
		c <- evt
	}
}

func (p *PubSub) Wait() {
	p.wg.Wait()
}
