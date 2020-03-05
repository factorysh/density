package scheduler

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	_task "github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16 * 1024,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)
	wait := sync.WaitGroup{}
	wait.Add(1)
	task := &_task.Task{
		Start:           time.Now(),
		MaxExectionTime: 30 * time.Second,
		Action: func(context.Context) error {
			fmt.Println("Action A")
			time.Sleep(200 * time.Millisecond)
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
	assert.Len(t, s.readyToGo(), 0)

	// Second part

	wait.Add(2)
	actions := make([]int, 0)
	for _, task := range []*_task.Task{
		&_task.Task{
			Start:           time.Now(),
			CPU:             2,
			RAM:             512,
			MaxExectionTime: 30 * time.Second,
			Action: func(context.Context) error {
				fmt.Println("Action B")
				time.Sleep(400 * time.Millisecond)
				actions = append(actions, 1)
				wait.Done()
				return nil
			},
		},
		&_task.Task{
			Start:           time.Now(),
			CPU:             3,
			RAM:             1024,
			MaxExectionTime: 30 * time.Second,
			Action: func(context.Context) error {
				fmt.Println("Action C")
				time.Sleep(300 * time.Millisecond)
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
	flushed := s.Flush(0)
	assert.Equal(t, 3, flushed)
}

func TestFlood(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16 * 1024,
	})
	actions := make([]uuid.UUID, 0)
	wait := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()
	size := 30
	for i := 0; i < size; i++ {
		wait.Add(1)
		s.Add(&_task.Task{
			Start:           time.Now(),
			CPU:             rand.Intn(4) + 1,
			RAM:             (rand.Intn(16) + 1) * 256,
			MaxExectionTime: 30 * time.Second,
			Action: func(ctx context.Context) error {
				t, _ := ctx.Value("task").(*_task.Task)
				time.Sleep(time.Duration(int64(rand.Intn(250)+1)) * time.Millisecond)
				fmt.Println("Done ", t.Id)
				actions = append(actions, t.Id)
				wait.Done()
				return nil
			},
		})
	}
	wait.Wait()
	fmt.Println(len(actions), actions)
	assert.Len(t, actions, size)
}

func TestTimeout(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16 * 1024,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	wait := sync.WaitGroup{}
	wait.Add(1)
	var action string
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             2,
		RAM:             256,
		MaxExectionTime: 1 * time.Second,
		Action: func(ctx context.Context) error {
			select {
			case <-time.After(2 * time.Second):
				fmt.Println("2s")
				action = "waiting"
			case <-ctx.Done():
				fmt.Println("canceled")
				action = "canceled"
			}
			wait.Done()
			return nil
		},
	}
	_, err := s.Add(task)
	assert.NoError(t, err)
	wait.Wait()
	assert.Equal(t, "canceled", action)
	assert.Len(t, s.tasks, 1)
	for _, tt := range s.tasks {
		assert.NotEqual(t, _task.Waiting, tt.Status)
		assert.NotEqual(t, _task.Running, tt.Status)
	}
}

func TestCancel(t *testing.T) {
	s := New(Playground{
		CPU: 4,
		RAM: 16 * 1024,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	wait := sync.WaitGroup{}
	wait.Add(1)
	var action string
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             2,
		RAM:             256,
		MaxExectionTime: 31 * time.Second,
		Action: func(ctx context.Context) error {
			select {
			case <-time.After(2 * time.Second):
				fmt.Println("2s")
				action = "waiting"
			case <-ctx.Done():
				fmt.Println("canceled")
				action = "canceled"
			}
			wait.Done()
			return nil
		},
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	err = s.Cancel(id)
	assert.NoError(t, err)
	wait.Wait()
	assert.Equal(t, 1, s.Length())
	assert.Equal(t, "canceled", action)
}
