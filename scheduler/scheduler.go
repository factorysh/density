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
	playGround  Playground
	tasks       map[uuid.UUID]*Task
	lock        sync.RWMutex
	waitingLock sync.RWMutex
	events      chan int
	newTasks    chan *Task
	CPU         int
	RAM         int
	processes   int
	waiting     bool
}

func New(playground Playground) *Scheduler {
	return &Scheduler{
		playGround: playground,
		tasks:      make(map[uuid.UUID]*Task),
		lock:       sync.RWMutex{},
		events:     make(chan int, 100),
		newTasks:   make(chan *Task, 100),
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
		return id, err
	}
	task.Id = id
	s.newTasks <- task
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
		select {
		case task := <-s.newTasks:
			s.lock.Lock()
			s.tasks[task.Id] = task
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
			}
			go func() {
				time.Sleep(sleep)
				s.events <- 1
			}()
		} else { // Something todo
			l.WithField("todos", len(todos)).Info()
			s.lock.Lock()
			chosen := todos[0]
			s.CPU -= chosen.CPU
			s.RAM -= chosen.RAM
			s.processes++
			log.WithField("cpu", s.CPU).WithField("ram", s.RAM).WithField("process", s.processes).Info()
			go func(task *Task, cpu, ram int) {
				chosen.Action(context.WithValue(context.TODO(), "task", task))
				s.lock.Lock()
				s.CPU += cpu
				s.RAM += ram
				s.processes--
				s.lock.Unlock()
				s.events <- 1 // a slot is now free, let's try to full it
			}(chosen, chosen.CPU, chosen.RAM)
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
	i := 0
	s.lock.RLock()
	defer s.lock.RUnlock()
	tasks := make(TaskByStart, len(s.tasks))
	for _, task := range s.tasks {
		tasks[i] = task
		i++
	}
	sort.Sort(tasks)
	return tasks[0]
}
