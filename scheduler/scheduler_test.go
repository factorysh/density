package scheduler

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/factorysh/batch-scheduler/runner/compose"
	compose_runner "github.com/factorysh/batch-scheduler/runner/compose"
	"github.com/factorysh/batch-scheduler/store"
	"github.com/factorysh/batch-scheduler/task"
	_task "github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type DummyRunner struct {
}

func (d *DummyRunner) Up(ctx context.Context, _task *task.Task) error {
	return _task.Action.Run(ctx, "/tmp/", nil)
}

func TestScheduler(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), compose_runner.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)
	task := &_task.Task{
		Owner:           "test",
		Start:           time.Now(),
		MaxExectionTime: 30 * time.Second,
		Action: &_task.DummyAction{
			Name: "Action A",
			Wait: 10,
		},
		CPU: 2,
		RAM: 256,
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	wait := &sync.WaitGroup{}
	wait.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		events := s.Pubsub.Subscribe(ctx)
		for {
			event := <-events
			if event.Id == id && event.Action == "done" {
				wait.Done()
				return
			}
		}
	}()
	list := s.List()
	assert.Equal(t, 1, len(list))
	filtered := s.Filter("test")
	assert.Equal(t, 1, len(filtered))
	fmt.Println("id", id)
	wait.Wait()
	assert.Len(t, s.readyToGo(), 0)

	// Second part

	wait.Add(2)
	actions := make([]int, 0)
	ids := make([]uuid.UUID, 0)
	for _, task := range []*_task.Task{
		{
			Start:           time.Now(),
			CPU:             2,
			RAM:             512,
			MaxExectionTime: 30 * time.Second,
			Action: &_task.DummyAction{
				Name: "Action B",
				Wait: 400,
			},
		},
		{
			Start:           time.Now(),
			CPU:             3,
			RAM:             1024,
			MaxExectionTime: 30 * time.Second,
			Action: &_task.DummyAction{
				Name: "Action C",
				Wait: 300,
			},
		},
	} {
		id, err = s.Add(task)
		assert.NoError(t, err)
		ids = append(ids, id)
	}
	go func() {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		events := s.Pubsub.Subscribe(ctx)
		i := 2
		for {
			event := <-events
			if event.Action == "done" {
				wait.Done()
				i--
			}
			if i == 0 {
				return
			}
		}
	}()
	wait.Wait()
	sort.Ints(actions)
	// TODO: FIXME
	// assert.Equal(t, []int{1, 2}, actions)
	flushed := s.Flush(0)
	assert.Equal(t, 3, flushed)
}

func TestFlood(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), compose_runner.New(dir), store.NewMemoryStore())
	wait := _task.NewWaiter()
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()
	a := _task.DummyAction{
		Name:    "Test Flood",
		Wait:    250,
		Counter: 0,
		Wg:      wait,
	}
	size := 30
	for i := 0; i < size; i++ {
		wait.Add(1)
		s.Add(&_task.Task{
			Start:           time.Now(),
			CPU:             rand.Intn(4) + 1,
			RAM:             (rand.Intn(16) + 1) * 256,
			MaxExectionTime: 30 * time.Second,
			Action:          &a,
		})
	}
	wait.Wait()
	fmt.Println(a.Counter)
	assert.Equal(t, a.Counter, int64(size))
}

func TestTimeout(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), compose_runner.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	wait := _task.NewWaiter()
	a := _task.DummyAction{
		Name:        "Test Timeout",
		WithTimeout: true,
		Wg:          wait,
	}
	wait.Add(1)
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             2,
		RAM:             256,
		MaxExectionTime: 1 * time.Second,
		Action:          &a,
	}
	_, err = s.Add(task)
	assert.NoError(t, err)
	wait.Wait()
	assert.Equal(t, "canceled", a.Status)
	assert.Len(t, s.tasks, 1)
	s.tasks.ForEach(func(tt *_task.Task) error {
		assert.NotEqual(t, _task.Waiting, tt.Status)
		assert.NotEqual(t, _task.Running, tt.Status)
		return nil
	})
}

func TestCancel(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), compose_runner.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	wait := _task.NewWaiter()
	a := _task.DummyAction{
		Name:        "Test Timeout",
		WithTimeout: true,
		Wg:          wait,
	}
	wait.Add(1)
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             2,
		RAM:             256,
		MaxExectionTime: 31 * time.Second,
		Action:          &a,
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	err = s.Cancel(id)
	assert.NoError(t, err)
	wait.Wait()
	assert.Equal(t, 1, s.Length())
	assert.Equal(t, "canceled", a.Status)
}

func TestExec(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), compose.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	wait := _task.NewWaiter()
	a := _task.DummyAction{
		Name:        "Test Exec",
		WithCommand: true,
		Wg:          wait,
	}
	wait.Add(1)
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             1,
		RAM:             64,
		MaxExectionTime: 1 * time.Second,
		Action:          &a,
	}
	_, err = s.Add(task)
	assert.NoError(t, err)
	wait.Wait()
	assert.NotEqual(t, 0, a.ExitCode)
}
