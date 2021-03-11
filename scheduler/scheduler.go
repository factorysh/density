package scheduler

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/factorysh/density/pubsub"
	_store "github.com/factorysh/density/store"
	"github.com/factorysh/density/task"
	_task "github.com/factorysh/density/task"
	_run "github.com/factorysh/density/task/run"
	_status "github.com/factorysh/density/task/status"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	resources *Resources
	tasks     *JSONStore
	lock      sync.RWMutex
	events    chan interface{}
	stop      chan bool
	runner    Runner
	Pubsub    *pubsub.PubSub
	dataDir   string
}

type Runner interface {
	Up(*task.Task) (_run.Run, error)
	GetHome() string
}

func New(resources *Resources, runner Runner, store _store.Store) *Scheduler {
	return &Scheduler{
		resources: resources,
		tasks:     &JSONStore{store},
		events:    make(chan interface{}),
		stop:      make(chan bool),
		runner:    runner,
		Pubsub:    pubsub.NewPubSub(),
	}
}

func (s *Scheduler) Add(task *_task.Task) (uuid.UUID, error) {
	if task.Id != uuid.Nil {
		return uuid.Nil, errors.New("I am choosing the uuid, not you")
	}
	err := s.resources.Check(task.CPU, task.RAM)
	if err != nil {
		return uuid.Nil, err
	}
	if task.MaxExectionTime <= 0 {
		return uuid.Nil, errors.New("MaxExectionTime must be > 0")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return uuid.Nil, err
	}
	task.Id = id
	task.Status = _status.Waiting
	task.Mtime = time.Now()
	err = s.tasks.Put(task)
	if err != nil {
		return uuid.Nil, err
	}
	task.Cancel = func() {
		task.Status = _status.Canceled
	}
	s.events <- new(interface{})
	s.Pubsub.Publish(pubsub.Event{
		Action: "added",
		Id:     id,
	})
	return id, nil
}

// Load will fetch jobs data and status from storage
func (s *Scheduler) Load() error {
	// to remove tasks
	garbage := make([]*_task.Task, 0)
	// to update tasks
	update := make([]*_task.Task, 0)

	err := s.tasks.ForEach(func(t *_task.Task) error {
		// remember old status
		old := t.Status
		// fresh status
		var fresh _status.Status
		var status _run.Status
		var exit int
		var err error

		// fetch status
		if t.Run != nil {
			status, exit, err = t.Run.Status()
			if err != nil {
				return err
			}
		} else {
			status = _run.Unkown
		}

		// map runner status to task status
		switch status {
		case _run.Running:
			fresh = _status.Running
			// exec task will consume ressources, attach a watcher to exesting task without relaunching the entire task
			s.execTask(t)
		case _run.Dead:
			fresh = _status.Error
		case _run.Exited:
			if exit != 0 {
				fresh = _status.Error
			} else {
				fresh = _status.Done
			}
		default:
			// gc the ones not found
			garbage = append(garbage, t)
		}

		// if status mismatch, update
		if old != fresh {
			t.Status = fresh
			update = append(update, t)
		}
		return err
	})

	for _, t := range garbage {
		err := s.tasks.Delete(t.Id)
		if err != nil {
			return err
		}
	}

	for _, t := range update {
		err := s.tasks.Put(t)
		if err != nil {
			return err
		}
	}

	return err
}

// Start is the main loop
func (s *Scheduler) Start(ctx context.Context) {
	// FIXME, find all detached running tasks in s.tasks
	for {
		select {
		case <-s.events:
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		}
		l := log.WithField("tasks", s.tasks.Length())
		todos := s.readyToGo()
		l = l.WithField("todos", len(todos))
		if len(todos) == 0 { // nothing is ready  just wait
			now := time.Now()
			n := s.next()
			var sleep time.Duration
			if n == nil {
				sleep = 1 * time.Second
			} else {
				sleep = now.Sub(n.Start)
				l = l.WithField("task", n.Id)
			}
			l.WithField("sleep", sleep).Info("Waiting")
			time.AfterFunc(sleep, func() {
				s.events <- new(interface{})
			})
		} else { // Something todo
			s.execTask(todos[0])
		}
	}
}

// Exec chosen task
func (s *Scheduler) execTask(chosen *_task.Task) {
	s.lock.Lock()
	ctxResources, cancelResources := context.WithCancel(context.TODO())
	s.resources.Consume(ctxResources, chosen.CPU, chosen.RAM)
	log.WithFields(log.Fields{
		"cpu":     s.resources.cpu,
		"ram":     s.resources.ram,
		"process": s.resources.processes,
	}).Info()
	run, err := s.runner.Up(chosen)
	if err != nil {
		chosen.Status = _status.Error
		cancelResources()
		log.WithError(err).Error()
		s.tasks.Put(chosen)
		s.lock.Unlock()
		return
	}
	chosen.Status = _status.Running
	chosen.Run = run
	s.tasks.Put(chosen)

	ctx, cancel := context.WithTimeout(context.TODO(), chosen.MaxExectionTime)

	cleanup := func() {
		cancel()
		cancelResources()
	}
	s.Pubsub.Publish(pubsub.Event{
		Action: chosen.Status.String(),
		Id:     chosen.Id,
	})
	s.lock.Unlock()
	go func(ctx context.Context, task *task.Task, run _run.Run, cleanup func()) {
		defer cleanup()
		status, err := run.Wait(ctx)
		if err != nil {
			log.WithError(err).Error()
		}
		task.Status = status
		s.Pubsub.Publish(pubsub.Event{
			Action: task.Status.String(),
			Id:     task.Id,
		})
		s.tasks.Put(task)
		s.events <- new(interface{}) // a slot is now free, let's try to full it
	}(ctx, chosen, run, cleanup)
}

// List all the tasks associated with this scheduler
func (s *Scheduler) List() []*_task.Task {
	tasks := make([]*_task.Task, 0)

	s.tasks.ForEach(func(t *_task.Task) error {
		tasks = append(tasks, t)
		return nil
	})

	return tasks
}

// Filter tasks for a specific owner
func (s *Scheduler) Filter(owner string) []*_task.Task {
	tasks := make([]*_task.Task, 0)

	s.lock.RLock()
	defer s.lock.RUnlock()

	s.tasks.ForEach(func(t *_task.Task) error {
		if t.Owner == owner {
			tasks = append(tasks, t)
		}
		return nil
	})

	return tasks
}

func (s *Scheduler) readyToGo() []*_task.Task {
	now := time.Now()
	tasks := make(_task.TaskByKarma, 0)
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.tasks.ForEach(func(task *_task.Task) error {
		// enough CPU, enough RAM, Start date is okay
		if task.Start.Before(now) && task.Status == _status.Waiting && s.resources.IsDoable(task.CPU, task.RAM) {
			tasks = append(tasks, task)
		}
		return nil
	})
	sort.Sort(tasks)
	return tasks
}

func (s *Scheduler) next() *_task.Task {
	if s.tasks.Length() == 0 {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	tasks := make(_task.TaskByStart, 0)
	s.tasks.ForEach(func(task *_task.Task) error {
		if task.Status == _status.Waiting {
			tasks = append(tasks, task)
		}
		return nil
	})
	if len(tasks) == 0 {
		return nil
	}
	sort.Sort(tasks)
	return tasks[0]
}

func (s *Scheduler) GetTask(id uuid.UUID) (*_task.Task, error) {
	return s.tasks.Get(id)
}

// Cancel a task
func (s *Scheduler) Cancel(id uuid.UUID) error {
	task, err := s.tasks.Get(id)
	if err != nil {
		return err
	}

	if task == nil {
		return errors.New("Unknown id")
	}

	if task.Status == _status.Canceled {
		return nil
	}

	// TODO: find a way to generate a Cancel method when getting the task from
	// the memory store
	task.Cancel = func() {
		if task.Run != nil {
			task.Run.Down()
		}
		task.Status = _status.Canceled
	}

	if task.Status == _status.Running {
		task.Cancel()
	}
	task.Mtime = time.Now()
	s.tasks.Put(task)
	return nil
}

// Length returns the number of Task
func (s *Scheduler) Length() int {
	return s.tasks.Length()
}

// Flush removes all done Tasks
func (s *Scheduler) Flush(age time.Duration) int {
	now := time.Now()
	i := 0
	s.tasks.DeleteWithClause(func(task *_task.Task) bool {
		if task.Status != _status.Running && task.Status != _status.Waiting && now.Sub(task.Mtime) > age {
			i++
			return true
		}
		return false
	})

	return i
}

// GetDataDir will return data dir for current runner
func (s *Scheduler) GetDataDir() string {
	return s.runner.GetHome()
}
