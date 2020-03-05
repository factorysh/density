package scheduler

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	playGround Playground
	tasks      map[uuid.UUID]*Task
	lock       sync.RWMutex
	events     chan int
	tasksTodo  chan *Task
	tasksDone  chan *Task
	CPU        int
	RAM        int
	processes  int
}

func New(playground Playground) *Scheduler {
	return &Scheduler{
		playGround: playground,
		tasks:      make(map[uuid.UUID]*Task),
		lock:       sync.RWMutex{},
		events:     make(chan int),
		tasksTodo:  make(chan *Task),
		tasksDone:  make(chan *Task),
		CPU:        playground.CPU,
		RAM:        playground.RAM,
	}
}

func (s *Scheduler) Add(task *Task) (uuid.UUID, error) {
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
	if task.CPU > s.playGround.CPU {
		return uuid.Nil, errors.New("Too much CPU is required")
	}
	if task.RAM > s.playGround.RAM {
		return uuid.Nil, errors.New("Too much RAM is required")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return uuid.Nil, err
	}
	task.Id = id
	task.Status = Waiting
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
			task.Status = Done
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
			chosen.Status = Running
			var ctx context.Context
			ctx, chosen.Cancel = context.WithTimeout(
				context.WithValue(context.TODO(), "task", chosen), chosen.MaxExectionTime)
			if chosen.Cancel == nil {
				panic("aaaah")
			}
			go func(ctx context.Context, task *Task) {
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

func (s *Scheduler) readyToGo() []*Task {
	now := time.Now()
	tasks := make(TaskByKarma, 0)
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, task := range s.tasks {
		// enough CPU, enough RAM, Start date is okay
		if task.Start.Before(now) && task.CPU <= s.CPU && task.RAM <= s.RAM && task.Status == Waiting {
			tasks = append(tasks, task)
		}
	}
	sort.Sort(tasks)
	return tasks
}

func (s *Scheduler) next() *Task {
	if len(s.tasks) == 0 {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	tasks := make(TaskByStart, 0)
	for _, task := range s.tasks {
		if task.Status == Waiting {
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
	if task.Status == Running {
		task.Cancel()
	}
	task.Status = Canceled
	return nil
}

func (s *Scheduler) Length() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.tasks)
}
