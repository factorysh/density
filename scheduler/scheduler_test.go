package scheduler

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/pubsub"
	"github.com/factorysh/density/runner"
	"github.com/factorysh/density/store"
	_task "github.com/factorysh/density/task"
	_ "github.com/factorysh/density/task/compose" // registering compose
	_status "github.com/factorysh/density/task/status"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func waitFor(ps *pubsub.PubSub, size int, clause func(evt pubsub.Event) bool) *sync.WaitGroup {
	wait := &sync.WaitGroup{}
	wait.Add(size)

	go func(size int) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		events := ps.Subscribe(ctx)
		for {
			event := <-events
			fmt.Println("wait for", event)
			if clause(event) {
				wait.Done()
				size--
				if size == 0 {
					return
				}
			} else {
				fmt.Println("Just an event ", event)
			}
		}
	}(size)
	return wait
}

func TestSchedulerStartStop(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), runner.New(dir, nil), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	assert.True(t, s.started)
	cancel()
	s.WaitStop()
	assert.False(t, s.started)
}

func TestScheduler(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), runner.New(dir, nil), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	assert.True(t, s.started)
	wait := waitFor(s.Pubsub, 1, func(event pubsub.Event) bool {
		return event.Action == "Done"
	})
	task := &_task.Task{
		Owner:           "test",
		Start:           time.Now(),
		MaxExectionTime: 5 * time.Second,
		Labels: map[string]string{
			"key": "value",
		},
		Action: &_task.DummyAction{
			Name: "Action A",
			Wait: 10 * time.Millisecond,
		},
		CPU: 2,
		RAM: 256,
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	list := s.List()
	assert.Len(t, list, 1)
	filtered := s.Filter("test", nil)
	assert.Len(t, filtered, 1)
	wait.Wait()
	assert.Len(t, s.readyToGo(), 0)
	filtered = s.Filter("test", nil)
	assert.Len(t, filtered, 1)
	filtered = s.Filter("test", map[string]string{
		"key": "value",
	})
	assert.Len(t, filtered, 1)

	assert.Equal(t, _status.Done, filtered[0].Status)

	// Second part

	wait = waitFor(s.Pubsub, 2, func(event pubsub.Event) bool {
		return event.Action == "Done"
	})
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
	wait.Wait()
	assert.Equal(t, 3, s.Length())
	flushed := s.Flush(0)
	assert.Equal(t, 3, flushed)

}

func TestFlood(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), runner.New(dir, nil), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	size := 30
	wait := waitFor(s.Pubsub, size, func(event pubsub.Event) bool {
		return event.Action == "Done"
	})
	for i := 0; i < size; i++ {
		s.Add(&_task.Task{
			Start:           time.Now(),
			CPU:             rand.Intn(4) + 1,
			RAM:             (rand.Intn(16) + 1) * 256,
			MaxExectionTime: 10 * time.Second,
			Action: &_task.DummyAction{
				Name:    fmt.Sprintf("Test Flood #%d", i),
				Wait:    250 * time.Millisecond,
				Counter: 0,
			},
		})
	}
	wait.Wait()
}

func TestTimeout(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), runner.New(dir, nil), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	wait := waitFor(s.Pubsub, 1, func(event pubsub.Event) bool {
		return event.Action == "Done"
	})
	a := _task.DummyAction{
		Name: "Test Timeout",
		Wait: 10,
	}
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
	// get task status from storage
	fromStorage, err := s.tasks.Get(task.Id)
	assert.NoError(t, err)
	assert.Equal(t, _status.Done, fromStorage.Status)
	assert.Equal(t, s.tasks.Length(), 1)
	s.tasks.ForEach(func(tt *_task.Task) error {
		assert.NotEqual(t, _status.Waiting, tt.Status)
		assert.NotEqual(t, _status.Running, tt.Status)
		return nil
	})
}

func TestLoad(t *testing.T) {
	// can't run in CI since access to docker host can be limited
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	// compose template
	composeTemplate := `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "sh -c 'sleep %d && echo world'"
x-batch:
  max_execution_time: 5s
`
	withCron := `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "sh -c 'sleep %d && echo world'"
x-batch:
  max_execution_time: 5s
  every: 1m
`

	c1 := compose.NewCompose()
	err := yaml.Unmarshal([]byte(fmt.Sprintf(composeTemplate, 0)), &c1)
	assert.NoError(t, err)

	c2 := compose.NewCompose()
	err = yaml.Unmarshal([]byte(fmt.Sprintf(composeTemplate, 5)), &c2)
	assert.NoError(t, err)

	c3 := compose.NewCompose()
	err = yaml.Unmarshal([]byte(fmt.Sprintf(withCron, 0)), &c3)
	assert.NoError(t, err)

	// setting up the scheduler
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	store, err := store.NewBoltStore(fmt.Sprintf("%s/bbolt.store", dir))
	assert.NoError(t, err)
	s := New(NewResources(4, 16*1024), runner.New(dir, nil), store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// init tasks
	tasks := [3]_task.Task{
		{
			Start:           time.Now(),
			CPU:             2,
			RAM:             256,
			MaxExectionTime: 3 * time.Second,
			Action:          c1,
		},
		{
			Start:           time.Now(),
			CPU:             2,
			RAM:             256,
			MaxExectionTime: 3 * time.Second,
			Action:          c2,
		},
		{
			Start:           time.Now(),
			CPU:             2,
			RAM:             256,
			MaxExectionTime: 3 * time.Second,
			Action:          c3,
		},
	}

	uuids := make([]uuid.UUID, 0)
	for _, task := range tasks {
		uuid, err := s.Add(&task)
		assert.NoError(t, err)
		uuids = append(uuids, uuid)
	}

	// stop the scheduler, like a restart
	cancel()
	s.WaitStop()

	s = New(NewResources(4, 16*1024), runner.New(dir, nil), store)
	// on restart, load is called to refresh state
	err = s.Load()
	assert.NoError(t, err)
	assert.Equal(t, 3, s.Length())

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// first one should be finished
	task, err := s.tasks.Get(uuids[0])
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, _status.Done, task.Status)

	time.Sleep(time.Duration(1 * time.Second))
	// second one shoud be running
	task, err = s.tasks.Get(uuids[1])
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, _status.Running, task.Status,
		fmt.Sprintf("task.Status should be Running, not %s", task.Status.String()))

	// FIXME: no reproducible
	// task, err = s.tasks.Get(uuids[2])
	// assert.NoError(t, err)
	// assert.NotNil(t, task)
	// assert.Equal(t, _status.Waiting, task.Status,
	// 	fmt.Sprintf("task.Status should be Waiting, not %s", task.Status.String()))

}

/*
func TestCancel(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(NewResources(4, 16*1024), runner.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	defer cancel()

	//wait := _task.NewWaiter()
	a := _task.DummyAction{
		Name:        "Test Cancel",
		WithTimeout: true,
		//Wg:          wait,
	}
	//wait.Add(1)
	task := &_task.Task{
		Start:           time.Now(),
		CPU:             2,
		RAM:             256,
		MaxExectionTime: 31 * time.Second,
		Action:          &a,
	}
	id, err := s.Add(task)
	assert.NoError(t, err)
	// wait for the task to be running
	time.Sleep(1 * time.Second)
	err = s.Cancel(id)
	assert.NoError(t, err)
	//wait.Wait()
	// wait for the action to run
	time.Sleep(1 * time.Second)
	// get task status from storage
	fromStorage, err := s.tasks.Get(task.Id)
	assert.NoError(t, err)
	assert.Equal(t, _task.Canceled, fromStorage.Status)
	assert.Equal(t, s.tasks.Length(), 1)
}

*/
