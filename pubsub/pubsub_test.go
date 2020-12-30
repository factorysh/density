package pubsub

import (
	"context"
	"fmt"
	"sync"
	"testing"

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
