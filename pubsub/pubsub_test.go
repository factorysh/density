package pubsub

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPubsub(t *testing.T) {
	ps := NewPubSub()
	wg := sync.WaitGroup{}
	size := 10
	wg.Add(size)
	cancels := make([]context.CancelFunc, 0)
	for i := 0; i < size; i++ {
		ctx, cancel := context.WithCancel(context.TODO())
		cancels = append(cancels, cancel)
		events := ps.Subscribe(ctx)

		go func() {
			for {
				event := <-events
				fmt.Println(event)
				wg.Done()
			}
		}()
	}
	assert.Len(t, ps.subscribers, size)
	ps.Publish(Event{})
	wg.Wait()

	for _, cancel := range cancels {
		cancel()
	}
	ps.Wait()
	assert.Len(t, ps.subscribers, 0)
}

func TestPubSubFlood(t *testing.T) {
	s := 1000
	m := 1000
	ps := NewPubSub()
	wg := sync.WaitGroup{}
	wg.Add(s * m)
	ready := sync.WaitGroup{}
	ready.Add(s)
	for i := 0; i < s; i++ {
		go func() {
			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()
			events := ps.Subscribe(ctx)
			ready.Done()
			for {
				event := <-events
				fmt.Println(event)
				wg.Done()
			}
		}()
	}
	ready.Wait()
	for i := 0; i < m; i++ {
		go func() {
			time.Sleep(time.Duration(rand.Float64()*10) * time.Millisecond)
			ps.Publish(Event{})
		}()
	}
	wg.Wait()
}
