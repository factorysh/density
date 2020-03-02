package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	wait := sync.WaitGroup{}
	wait.Add(1)
	task := &Task{
		Start: time.Now(),
		Action: func(context.Context) error {
			fmt.Println("Action")
			time.Sleep(5 * time.Second)
			wait.Done()
			return nil
		},
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	fmt.Println("id", id)
	wait.Wait()
	cancel()
}
