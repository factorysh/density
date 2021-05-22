package pubsub

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Event struct {
	Action string    `json:"action"`
	Id     uuid.UUID `json:"id"`
}

type PubSub struct {
	lock        *sync.RWMutex
	cpt         uint64
	subscribers map[uint64]chan Event
	wg          *sync.WaitGroup
}

func NewPubSub() *PubSub {
	return &PubSub{
		lock:        &sync.RWMutex{},
		cpt:         0,
		subscribers: make(map[uint64]chan Event),
		wg:          &sync.WaitGroup{},
	}
}

// Subscribe
func (p *PubSub) Subscribe(ctx context.Context) chan Event {
	p.lock.Lock()
	defer p.lock.Unlock()
	id := p.cpt
	p.cpt++
	p.subscribers[id] = make(chan Event, 1)
	p.wg.Add(1)
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

// Publish an event
func (p *PubSub) Publish(evt Event) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var worst time.Duration = 0
	// Warning, chans are blocking
	for _, c := range p.subscribers {
		now := time.Now()
		c <- evt
		delta := time.Since(now)
		if delta > worst {
			worst = delta
		}
	}
	log.WithField("event", evt).WithField("subscribers", len(p.subscribers)).WithField("worst duration", worst).Info("publish")
}

// Wait for all subscribers closing
func (p *PubSub) Wait() {
	p.wg.Wait()
}
