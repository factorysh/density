package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Task something to do
type Task struct {
	Start           time.Time
	MaxWaitTime     time.Duration
	MaxExectionTime time.Duration
	CPU             int
	RAM             int
	Action          Action
	Id              uuid.UUID
	Cancel          context.CancelFunc
	Status          Status
}

// Action does something
type Action func(context.Context) error

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
