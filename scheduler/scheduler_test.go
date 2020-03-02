package scheduler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16 * 1024,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	wait := sync.WaitGroup{}
	wait.Add(1)
	task := &Task{
		Start: time.Now(),
		Action: func(context.Context) error {
			fmt.Println("Action A")
			time.Sleep(2 * time.Second)
			wait.Done()
			return nil
		},
		CPU: 2,
		RAM: 256,
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	fmt.Println("id", id)
	wait.Wait()
	assert.Len(t, s.tasks, 0)

	// Second part

	wait.Add(2)
	actions := make([]int, 0)
	for _, task := range []*Task{
		&Task{
			Start: time.Now(),
			CPU:   2,
			RAM:   512,
			Action: func(context.Context) error {
				fmt.Println("Action B")
				time.Sleep(4 * time.Second)
				actions = append(actions, 1)
				wait.Done()
				return nil
			},
		},
		&Task{
			Start: time.Now(),
			CPU:   3,
			RAM:   1024,
			Action: func(context.Context) error {
				fmt.Println("Action C")
				time.Sleep(3 * time.Second)
				actions = append(actions, 2)
				wait.Done()
				return nil
			},
		},
	} {
		_, err = s.Add(task)
		assert.NoError(t, err)
	}
	wait.Wait()
	sort.Ints(actions)
	assert.Equal(t, []int{1, 2}, actions)
	cancel()
}
