package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task something to do
type Task struct {
	Start           time.Time          `json:"start"`              // Start time
	MaxWaitTime     time.Duration      `json:"max_wait_time"`      // Max wait time before starting Action
	MaxExectionTime time.Duration      `json:"max_execution_time"` // Max execution time
	CPU             int                `json:"cpu"`                // CPU quota
	RAM             int                `json:"ram"`                // RAM quota
	Action          Action             `json:"action"`             // Action is an abstract, the thing to do
	Id              uuid.UUID          `json:"id"`                 // Id
	Cancel          context.CancelFunc `json:"-"`                  // Cancel the action
	Status          Status             `json:"status"`             // Status
	Mtime           time.Time          `json:"mtime"`              // Modified time
	Owner           string             `json:"owner"`              // Owner
	Retry           int                `json:"retry"`              // Number of retry before crash
	Every           time.Duration      `json:"every"`              // Periodic execution. Exclusive with Cron
	Cron            string             `json:"cron"`               // Cron definition. Exclusive with Every
	resourceCancel  context.CancelFunc `json:"-"`
}

func (t *Task) UnmarshalJSON(b []byte) error {
	raw := struct {
		Start           time.Time       `json:"start"`              // Start time
		MaxWaitTime     time.Duration   `json:"max_wait_time"`      // Max wait time before starting Action
		MaxExectionTime time.Duration   `json:"max_execution_time"` // Max execution time
		CPU             int             `json:"cpu"`                // CPU quota
		RAM             int             `json:"ram"`                // RAM quota
		Action          json.RawMessage `json:"action"`             // Action is an abstract, the thing to do
		Id              uuid.UUID       `json:"id"`                 // Id
		Status          Status          `json:"status"`             // Status
		Mtime           time.Time       `json:"mtime"`              // Modified time
		Owner           string          `json:"owner"`              // Owner
		Retry           int             `json:"retry"`              // Number of retry before crash
		Every           time.Duration   `json:"every"`              // Periodic execution. Exclusive with Cron
		Cron            string          `json:"cron"`               // Cron definition. Exclusive with Every
	}{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	t.Start = raw.Start
	t.MaxWaitTime = raw.MaxWaitTime
	t.MaxExectionTime = raw.MaxExectionTime
	t.CPU = raw.CPU
	t.RAM = raw.RAM
	t.Id = raw.Id
	t.Status = raw.Status
	t.Mtime = raw.Mtime
	t.Owner = raw.Owner
	t.Retry = raw.Retry
	t.Every = raw.Every
	t.Cron = raw.Cron

	var d DummyAction
	err = json.Unmarshal(raw.Action, &d)
	if err != nil {
		return err
	}
	t.Action = &d

	return nil
}

// NewTask init a new task
func NewTask(o string, a Action) Task {
	t := New()
	t.Owner = o
	t.Action = a
	return *t
}

func New() *Task {
	return &Task{
		CPU:    1,
		RAM:    1,
		Status: Waiting,
		Mtime:  time.Now(),
	}
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
