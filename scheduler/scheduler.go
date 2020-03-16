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
	events    chan int
	tasksTodo chan *_task.Task
	tasksDone chan *_task.Task
	CPU       int
	RAM       int
	processes int
}

func New(resources *Resources) *Scheduler {
	return &Scheduler{
		resources: resources,
		tasks:     make(map[uuid.UUID]*_task.Task),
		lock:      sync.RWMutex{},
		events:    make(chan int),
		tasksTodo: make(chan *_task.Task),
		tasksDone: make(chan *_task.Task),
		CPU:       resources.TotalCPU,
		RAM:       resources.TotalRAM,
	}
}

func (s *Scheduler) Add(task *_task.Task) (uuid.UUID, error) {
	if task.Id != uuid.Nil {
		return uuid.Nil, errors.New("I am choosing the uuid, not you")
	}
	if task.CPU <= 0 {
		return uuid.Nil, errors.New("CPU must be > 0")
	}
	if task.RAM <= 0 {
		return uuid.Nil, errors.New("RAM must be > 0")
	}
	if task.MaxExectionTime <= 0 {
		return uuid.Nil, errors.New("MaxExectionTime must be > 0")
	}
	if task.CPU > s.resources.TotalCPU {
		return uuid.Nil, errors.New("Too much CPU is required")
	}
	if task.RAM > s.resources.TotalRAM {
		return uuid.Nil, errors.New("Too much RAM is required")
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
	s.tasksTodo <- task
	return id, nil
}

func (s *Scheduler) Start(ctx context.Context) {
	for {
		select {
		case <-s.tasksTodo:
		case task := <-s.tasksDone:
			s.lock.Lock()
			task.Mtime = time.Now()
			task.Status = _task.Done
			// FIXME lets garbage collector, later
			//delete(s.tasks, task.Id)
			s.CPU += task.CPU
			s.RAM += task.RAM
			s.processes--
			s.lock.Unlock()
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
				s.events <- 1
			}()
		} else { // Something todo
			s.lock.Lock()
			chosen := todos[0]
			s.CPU -= chosen.CPU
			s.RAM -= chosen.RAM
			s.processes++
			l.WithFields(log.Fields{
				"cpu":     s.CPU,
				"ram":     s.RAM,
				"process": s.processes,
			}).Info()
			chosen.Status = _task.Running
			chosen.Mtime = time.Now()
			var ctx context.Context
			ctx, chosen.Cancel = context.WithTimeout(
				context.WithValue(context.TODO(), "task", chosen), chosen.MaxExectionTime)
			if chosen.Cancel == nil {
				panic("aaaah")
			}
			go func(ctx context.Context, task *_task.Task) {
				if task.Cancel == nil {
					panic("oups")
				}
				defer task.Cancel()
				task.Action(ctx)
				s.tasksDone <- task // a slot is now free, let's try to full it
			}(ctx, chosen)
			s.lock.Unlock()
		}
	}
}

func (s *Scheduler) readyToGo() []*_task.Task {
	now := time.Now()
	tasks := make(_task.TaskByKarma, 0)
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, task := range s.tasks {
		// enough CPU, enough RAM, Start date is okay
		if task.Start.Before(now) && task.CPU <= s.CPU && task.RAM <= s.RAM && task.Status == _task.Waiting {
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
