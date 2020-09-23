package scheduler

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	_task "github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	resources *Resources
	tasks     map[uuid.UUID]*_task.Task
	lock      sync.RWMutex
	events    chan interface{}
}

func New(resources *Resources) *Scheduler {
	return &Scheduler{
		resources: resources,
		tasks:     make(map[uuid.UUID]*_task.Task),
		lock:      sync.RWMutex{},
		events:    make(chan interface{}),
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
	task.Status = _task.Waiting
	task.Mtime = time.Now()
	s.lock.Lock()
	s.tasks[task.Id] = task
	s.lock.Unlock()
	task.Cancel = func() {
		task.Status = _task.Canceled
	}
	s.events <- new(interface{})
	return id, nil
}

// Start is the main loop
func (s *Scheduler) Start(ctx context.Context) {
	for {
		select {
		case <-s.events:
		}
		l := log.WithField("tasks", len(s.tasks))
		todos := s.readyToGo()
		l = l.WithField("todos", len(todos))
		if len(todos) == 0 { // nothing is ready  just wait
			now := time.Now()
			n := s.next()
			var sleep time.Duration = 0
			if n == nil {
				sleep = 1 * time.Second
			} else {
				sleep = now.Sub(n.Start)
				l = l.WithField("task", n.Id)
			}
			l.WithField("sleep", sleep).Info("Waiting")
			go func() {
				time.Sleep(sleep)
				s.events <- new(interface{})
			}()
		} else { // Something todo
			s.lock.Lock()
			chosen := todos[0]
			ctxResources, cancelResources := context.WithCancel(context.TODO())
			s.resources.Consume(ctxResources, chosen.CPU, chosen.RAM)
			l.WithFields(log.Fields{
				"cpu":     s.resources.cpu,
				"ram":     s.resources.ram,
				"process": s.resources.processes,
			}).Info()
			chosen.Status = _task.Running
			chosen.Mtime = time.Now()
			ctx, cancel := context.WithTimeout(
				context.WithValue(context.TODO(), "task", chosen), chosen.MaxExectionTime)

			chosen.Cancel = func() {
				cancel()
				cancelResources()
				chosen.Status = _task.Canceled
				chosen.Mtime = time.Now()
			}
			go func(ctx context.Context, task *_task.Task) {
				defer task.Cancel()
				task.Action.Run(ctx)
				task.Status = _task.Done
				task.Mtime = time.Now()
				s.events <- new(interface{}) // a slot is now free, let's try to full it
			}(ctx, chosen)
			s.lock.Unlock()
		}
	}
}

// List all the tasks associated with this scheduler
func (s *Scheduler) List() []*_task.Task {
	tasks := []*_task.Task{}

	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, val := range s.tasks {
		tasks = append(tasks, val)
	}

	return tasks

}

// Filter tasks for a specific owner
func (s *Scheduler) Filter(owner string) []*_task.Task {
	tasks := []*_task.Task{}

	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, val := range s.tasks {
		if val.Owner == owner {
			tasks = append(tasks, val)
		}
	}

	return tasks
}

func (s *Scheduler) readyToGo() []*_task.Task {
	now := time.Now()
	tasks := make(_task.TaskByKarma, 0)
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, task := range s.tasks {
		// enough CPU, enough RAM, Start date is okay
		if task.Start.Before(now) && task.Status == _task.Waiting && s.resources.IsDoable(task.CPU, task.RAM) {
			tasks = append(tasks, task)
		}
	}
	sort.Sort(tasks)
	return tasks
}

func (s *Scheduler) next() *_task.Task {
	if len(s.tasks) == 0 {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	tasks := make(_task.TaskByStart, 0)
	for _, task := range s.tasks {
		if task.Status == _task.Waiting {
			tasks = append(tasks, task)
		}
	}
	if len(tasks) == 0 {
		return nil
	}
	sort.Sort(tasks)
	return tasks[0]
}

// Cancel a task
func (s *Scheduler) Cancel(id uuid.UUID) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return errors.New("Unknown id")
	}
	if task.Status == _task.Running {
		task.Cancel()
	}
	task.Status = _task.Canceled
	task.Mtime = time.Now()
	return nil
}

// Length returns the number of Task
func (s *Scheduler) Length() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.tasks)
}

func (s *Scheduler) Flush(age time.Duration) int {
	s.lock.Lock()
	defer s.lock.Unlock()
	now := time.Now()
	i := 0
	for id, task := range s.tasks {
		if task.Status != _task.Running && task.Status != _task.Waiting && now.Sub(task.Mtime) > age {
			delete(s.tasks, id)
			i++
		}
	}
	return i
}
