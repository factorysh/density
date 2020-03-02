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
	CPU        int
	RAM        int
}

func New(playground Playground) *Scheduler {
	return &Scheduler{
		playGround: playground,
		tasks:      make(map[uuid.UUID]*Task),
		lock:       sync.RWMutex{},
		events:     make(chan int),
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
	if task.CPU > s.playGround.CPU {
		return uuid.Nil, errors.New("Too much CPU is required")
	}
	if task.RAM > s.playGround.RAM {
		return uuid.Nil, errors.New("Too much RAM is required")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	id, err := uuid.NewRandom()
	if err != nil {
		return id, err
	}
	task.Id = id
	s.tasks[id] = task
	if len(s.tasks) == 1 { // tasks list was empty
		s.events <- 1
	}
	log.WithField("task", *task).Info("adding task")
	return id, nil
}

type TaskByStart []*Task

func (t TaskByStart) Len() int           { return len(t) }
func (t TaskByStart) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TaskByStart) Less(i, j int) bool { return t[i].Start.Before(t[j].Start) }

type TaskByKarma []*Task

func (t TaskByKarma) Len() int      { return len(t) }
func (t TaskByKarma) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t TaskByKarma) Less(i, j int) bool {
	return (t[i].RAM * t[i].CPU / int(int64(t[i].MaxExectionTime))) <
		(t[j].RAM * t[j].CPU / int(int64(t[j].MaxExectionTime)))
}

func (s *Scheduler) Start(ctx context.Context) {
	for {
		var sleep time.Duration
		var todos []*Task
		l := log.WithField("tasks", len(s.tasks))
		if len(s.tasks) == 0 {
			sleep = 10 * time.Second
		} else {
			todos = s.readyToGo()
			if len(todos) == 0 { // nothing is ready  just wait
				n := s.next()
				now := time.Now()
				sleep = n.Start.Sub(now)
				if sleep <= 0 {
					// FIXME
					l.WithField("sleep", sleep).Warn()
					sleep = 10 * time.Second
				}
			}
		}
		if sleep != 0 {
			l.Info("Sleeping ", sleep)
			select {
			case <-time.After(sleep):
				continue
			case <-s.events:
				continue
			}
		} else { // Something todo
			l.WithField("todos", len(todos)).Info()
			s.lock.Lock()
			chosen := todos[0]
			s.CPU -= chosen.CPU
			s.RAM -= chosen.RAM
			go func() {
				chosen.Action(context.Background())
				s.lock.Lock()
				s.CPU += chosen.CPU
				s.RAM += chosen.RAM
				s.lock.Unlock()
			}()
			delete(s.tasks, chosen.Id)
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
		if task.Start.Before(now) && task.CPU <= s.CPU && task.RAM <= s.RAM {
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
	tasks := make(TaskByStart, len(s.tasks))
	i := 0
	for _, task := range s.tasks {
		tasks[i] = task
		i++
	}
	sort.Sort(tasks)
	return tasks[0]
}
