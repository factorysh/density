package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ActionsRegistry register all Action implementation
var ActionsRegistry map[string]func() Action

func init() {
	if ActionsRegistry == nil {
		ActionsRegistry = make(map[string]func() Action)
	}
	ActionsRegistry["dummy"] = func() Action {
		return &DummyAction{
			waiters: make([]chan interface{}, 0),
		}
	}
}

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
	Environments    map[string]string  `json:"environments,omitempty"`
	resourceCancel  context.CancelFunc `json:"-"`
	Run             Run                `json:"run"`
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type RawTask struct {
	Start           time.Time                  `json:"start"`              // Start time
	MaxWaitTime     Duration                   `json:"max_wait_time"`      // Max wait time before starting Action
	MaxExectionTime Duration                   `json:"max_execution_time"` // Max execution time
	CPU             int                        `json:"cpu"`                // CPU quota
	RAM             int                        `json:"ram"`                // RAM quota
	Action          map[string]json.RawMessage `json:"action"`             // Action is an abstract, the thing to do
	Id              uuid.UUID                  `json:"id"`                 // Id
	Status          Status                     `json:"status"`             // Status
	Mtime           time.Time                  `json:"mtime"`              // Modified time
	Owner           string                     `json:"owner"`              // Owner
	Retry           int                        `json:"retry"`              // Number of retry before crash
	Every           time.Duration              `json:"every"`              // Periodic execution. Exclusive with Cron
	Cron            string                     `json:"cron"`               // Cron definition. Exclusive with Every
	Environments    map[string]string          `json:"environments,omitempty"`
}

func (t *Task) UnmarshalJSON(b []byte) error {
	var raw RawTask
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	l := len(raw.Action)
	switch {
	case l == 0:
		t.Action = nil
	case l > 1:
		return fmt.Errorf("Two many actions %d", len(raw.Action))
	default:
		for k, v := range raw.Action {
			factory, ok := ActionsRegistry[k]
			if !ok {
				return fmt.Errorf("Unregistered action : %s", k)
			}
			t.Action = factory()
			err := json.Unmarshal(v, t.Action)
			if err != nil {
				return err
			}
		}
	}
	t.Start = raw.Start
	t.MaxWaitTime = time.Duration(raw.MaxWaitTime)
	t.MaxExectionTime = time.Duration(raw.MaxExectionTime)
	t.CPU = raw.CPU
	t.RAM = raw.RAM
	t.Id = raw.Id
	t.Status = raw.Status
	t.Mtime = raw.Mtime
	t.Owner = raw.Owner
	t.Retry = raw.Retry
	t.Every = raw.Every
	t.Cron = raw.Cron
	t.Environments = raw.Environments

	return nil
}

func (t *Task) MarshalJSON() ([]byte, error) {
	raw := RawTask{
		Start:           t.Start,
		MaxWaitTime:     Duration(t.MaxWaitTime),
		MaxExectionTime: Duration(t.MaxExectionTime),
		CPU:             t.CPU,
		RAM:             t.RAM,
		Id:              t.Id,
		Status:          t.Status,
		Mtime:           t.Mtime,
		Owner:           t.Owner,
		Retry:           t.Retry,
		Every:           t.Every,
		Cron:            t.Cron,
		Environments:    t.Environments,
		Action:          make(map[string]json.RawMessage),
	}
	if t.Action != nil {
		rawAction, err := json.Marshal(t.Action)
		if err != nil {
			return nil, err
		}
		name := t.Action.RegisteredName()
		raw.Action[name] = rawAction
	}
	return json.Marshal(raw)
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
