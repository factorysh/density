package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/factorysh/density/pubsub"
	_store "github.com/factorysh/density/store"
	"github.com/factorysh/density/task"
	_run "github.com/factorysh/density/task/run"
	_status "github.com/factorysh/density/task/status"
	"github.com/factorysh/density/todo"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	resources            *Resources
	tasks                *JSONStore
	lock                 sync.RWMutex
	somethingNewHappened *todo.Todo
	stop                 chan bool
	runner               Runner
	Pubsub               *pubsub.PubSub
	stopping             *sync.WaitGroup
	started              bool
}

type Runner interface {
	Up(*task.Task) (_run.Run, error)
	GetHome() string
}

func New(resources *Resources, runner Runner, store _store.Store) *Scheduler {
	return &Scheduler{
		resources:            resources,
		tasks:                &JSONStore{store},
		somethingNewHappened: todo.New(),
		stop:                 make(chan bool),
		runner:               runner,
		Pubsub:               pubsub.NewPubSub(),
		stopping:             &sync.WaitGroup{},
		started:              false,
	}
}

// Add a new task
func (s *Scheduler) Add(task *task.Task) (uuid.UUID, error) {
	if !s.started {
		return uuid.Nil, errors.New("Scheduler is not started")
	}
	if task.Id != uuid.Nil {
		return uuid.Nil, errors.New("don't choose your UUID, it's my job")
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
	s.somethingNewHappened.Ping()
	s.Pubsub.Publish(pubsub.Event{
		Action: "added",
		Id:     id,
	})
	return id, nil
}

// Load will fetch jobs data and status from storage
func (s *Scheduler) Load() error {
	if s.started {
		return errors.New("don't load a started scheduler")
	}
	// to remove tasks
	garbage := make([]*task.Task, 0)
	// to update tasks
	update := make([]*task.Task, 0)

	err := s.tasks.ForEach(func(t *task.Task) error {
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
		case _run.Unkown:
			// FIXME
			update = append(update, t)
		default:
			// gc the ones not found
			log.WithField("status", status).Info("Garbage")
			garbage = append(garbage, t)
		}

		// if status mismatch, update
		if old != fresh {
			t.Status = fresh
			update = append(update, t)
		} else if t.HasCron() && t.Status != _status.Running {
			t.Status = _status.Waiting
			t.PrepareReschedule()
			update = append(update, t)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, t := range garbage {
		log.WithField("id", t.Id).Info("removed while store load")
		err := s.tasks.Delete(t.Id)
		if err != nil {
			return err
		}
	}

	for _, t := range update {
		log.WithField("id", t.Id).Info("Back in main loop while store load")
		err := s.tasks.Put(t)
		if err != nil {
			return err
		}
	}

	s.oneLoop()
	return nil
}

// Start is the main loop, non blocking
func (s *Scheduler) Start(ctx context.Context) {
	if s.started {
		panic("Start once")
	}
	s.stopping.Add(1)
	// FIXME, find all detached running tasks in s.tasks
	log.Info("Starting main loop")
	go func() {
		for {
			select { // waiting for a trigger
			case <-s.stop:
				err := s.tasks.store.Sync()
				s.stopping.Done()
				if err != nil {
					log.WithError(err).Error("Stop and sync")
				}
				s.started = false
				log.Info("Scheduler loop is stopped")
				return // stop the loop
			case <-ctx.Done():
				err := s.tasks.store.Sync()
				if err != nil {
					log.WithError(err).Error("Context.Done and sync")
				}
				s.started = false
				s.stopping.Done()
				log.Info("Scheduler start context is done, loop is stopped.")
				return // stop the loop
			case <-s.somethingNewHappened.Wait():
			}
			s.oneLoop()
		}
	}()
	s.started = true
}

func (s *Scheduler) oneLoop() {
	chrono := time.Now()
	todos := s.readyToGo()
	l := log.WithField("tasks", s.tasks.Length()).WithField("todos", len(todos))
	defer l.WithField("chrono", time.Since(chrono)).Debug("Main loop iteration")
	if len(todos) > 0 { // Something todo
		s.execTask(todos[0])
		s.somethingNewHappened.Done()
		s.somethingNewHappened.Ping() // is there any // tasks waiting?
		return
	}
	// nothing is ready just wait
	now := time.Now()
	n := s.next()
	if n != nil {
		sleep := n.Start.Sub(now)
		l.WithField("task", n.Id).WithField("sleep", sleep).Info("Waiting")
		time.AfterFunc(sleep, func() {
			s.somethingNewHappened.Ping()
		})
	} // else no future
	s.somethingNewHappened.Done()
}

// Exec chosen task
func (s *Scheduler) execTask(chosen *task.Task) {
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
	chosen.Start = time.Now()
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
		if task.HasCron() {
			task.Status = _status.Waiting
			task.PrepareReschedule()
		}
		s.tasks.Put(task)
		s.somethingNewHappened.Ping() // a slot is now free, let's try to full it
	}(ctx, chosen, run, cleanup)
}

// List all the tasks associated with this scheduler
func (s *Scheduler) List() []*task.Task {
	tasks := make([]*task.Task, 0)

	s.tasks.ForEach(func(t *task.Task) error {
		tasks = append(tasks, t)
		return nil
	})

	return tasks
}

// Filter tasks for a specific owner
func (s *Scheduler) Filter(owner string, labels map[string]string) []*task.Task {
	tasks := make([]*task.Task, 0)

	s.lock.RLock()
	defer s.lock.RUnlock()

	s.tasks.ForEach(func(t *task.Task) error {
		if owner != "" && t.Owner != owner {
			return nil
		}
		for key, value := range labels {
			taskValue, found := t.Labels[key]
			if !found || taskValue != value {
				return nil
			}
		}
		tasks = append(tasks, t)
		return nil
	})

	return tasks
}

func (s *Scheduler) readyToGo() []*task.Task {
	now := time.Now()
	tasks := make(task.TaskByKarma, 0)
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.tasks.ForEach(func(task *task.Task) error {
		// enough CPU, enough RAM, Start date is okay
		if task.Start.Before(now) && task.Status == _status.Waiting && s.resources.IsDoable(task.CPU, task.RAM) {
			tasks = append(tasks, task)
		}
		return nil
	})
	sort.Sort(tasks)
	return tasks
}

func (s *Scheduler) next() *task.Task {
	if s.tasks.Length() == 0 {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	tasks := make(task.TaskByStart, 0)
	s.tasks.ForEach(func(task *task.Task) error {
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

func (s *Scheduler) GetTask(id uuid.UUID) (*task.Task, error) {
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
	return s.tasks.Put(task)
}

// Delete a task
func (s *Scheduler) Delete(id uuid.UUID) error {
	task, err := s.tasks.Get(id)
	if err != nil {
		return err
	}

	if task == nil {
		return fmt.Errorf("unknown id %s", id.String())
	}

	if task.Status == _status.Running && task.Run != nil {
		task.Run.Down()
	}

	return s.tasks.Delete(id)
}

// Length returns the number of Task
func (s *Scheduler) Length() int {
	return s.tasks.Length()
}

// Flush removes all done Tasks
func (s *Scheduler) Flush(age time.Duration) int {
	now := time.Now()
	i := 0
	s.tasks.DeleteWithClause(func(task *task.Task) bool {
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

func (s *Scheduler) WaitStop() {
	s.stopping.Wait()
}
