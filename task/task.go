package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/factorysh/density/task/action"
	_run "github.com/factorysh/density/task/run"
	"github.com/factorysh/density/task/status"
	"github.com/google/uuid"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// ActionsRegistry register all Action implementation
var ActionsRegistry map[string]func() action.Action

// RunRegistry register all Run implementation
var RunRegistry map[string]func() _run.Run

// UUID indentifier for tasks
const UUID = "uuid"

var Parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func init() {
	if ActionsRegistry == nil {
		ActionsRegistry = make(map[string]func() action.Action)
	}
	ActionsRegistry["dummy"] = func() action.Action {
		return &DummyAction{
			waiters: make([]chan interface{}, 0),
		}
	}

	if RunRegistry == nil {
		RunRegistry = make(map[string]func() _run.Run)
	}
	RunRegistry["dummy"] = func() _run.Run {
		return &DummyRun{
			da: nil,
		}
	}

}

// IsLabelValid is a simple function used to check validity of both key and value pairs in labels
var IsLabelValid = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`).MatchString

// Task something to do
type Task struct {
	Start           time.Time          `json:"start"`              // Start time
	MaxWaitTime     time.Duration      `json:"max_wait_time"`      // Max wait time before starting Action
	MaxExectionTime time.Duration      `json:"max_execution_time"` // Max execution time
	CPU             int                `json:"cpu"`                // CPU quota
	RAM             int                `json:"ram"`                // RAM quota
	Action          action.Action      `json:"action"`             // Action is an abstract, the thing to do
	Id              uuid.UUID          `json:"id"`                 // Id
	Cancel          context.CancelFunc `json:"-"`                  // Cancel the action
	Status          status.Status      `json:"status"`             // Status
	Mtime           time.Time          `json:"mtime"`              // Modified time
	Owner           string             `json:"owner"`              // Owner
	Retry           int                `json:"retry"`              // Number of retry before crash
	Every           time.Duration      `json:"every"`              // Periodic execution. Exclusive with Cron
	Cron            string             `json:"cron"`               // Cron definition. Exclusive with Every
	Environments    map[string]string  `json:"environments,omitempty"`
	resourceCancel  context.CancelFunc `json:"-"`
	Run             _run.Run           `json:"run"`
	Labels          map[string]string  `json:"labels"`
}

// Resp represent a task that can be send directly on the wire
type Resp struct {
	Start           time.Time         `json:"start"`              // Start time
	MaxWaitTime     time.Duration     `json:"max_wait_time"`      // Max wait time before starting Action
	MaxExectionTime time.Duration     `json:"max_execution_time"` // Max execution time
	CPU             int               `json:"cpu"`                // CPU quota
	RAM             int               `json:"ram"`                // RAM quota
	Id              uuid.UUID         `json:"id"`                 // Id
	Status          status.Status     `json:"status"`             // Status
	Mtime           time.Time         `json:"mtime"`              // Modified time
	Owner           string            `json:"owner"`              // Owner
	Retry           int               `json:"retry"`              // Number of retry before crash
	Every           time.Duration     `json:"every"`              // Periodic execution. Exclusive with Cron
	Cron            string            `json:"cron"`               // Cron definition. Exclusive with Every
	Environments    map[string]string `json:"environments,omitempty"`
	Run             _run.Data         `json:"run"`
	Labels          map[string]string `json:"labels"`
}

// ToTaskResp will Convert a Task to TaskResp
func (t *Task) ToTaskResp() Resp {

	return Resp{
		Start:           t.Start,
		MaxWaitTime:     t.MaxWaitTime,
		MaxExectionTime: t.MaxExectionTime,
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
		Run:             t.Run.Data(),
		Labels:          t.Labels,
	}

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
	Status          status.Status              `json:"status"`             // Status
	Mtime           time.Time                  `json:"mtime"`              // Modified time
	Owner           string                     `json:"owner"`              // Owner
	Retry           int                        `json:"retry"`              // Number of retry before crash
	Every           time.Duration              `json:"every"`              // Periodic execution. Exclusive with Cron
	Cron            string                     `json:"cron"`               // Cron definition. Exclusive with Every
	Environments    map[string]string          `json:"environments,omitempty"`
	Run             map[string]json.RawMessage `json:"run"`
	Labels          map[string]string          `json:"labels"`
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
	m := len(raw.Run)
	switch {
	case m == 0:
		t.Run = nil
	case m > 1:

	default:
		for k, v := range raw.Run {
			factory, ok := RunRegistry[k]
			if !ok {
				return fmt.Errorf("Unregistered run : %s", k)
			}
			t.Run = factory()
			err := json.Unmarshal(v, t.Run)
			if err != nil {
				return err
			}
		}
	}
	// Ensure cron is valid
	if t.Cron != "" {
		_, err = Parser.Parse(t.Cron)
		if err != nil {
			return fmt.Errorf("error when parsing cron string: %v", err)
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
	t.Labels = raw.Labels

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
		Run:             make(map[string]json.RawMessage),
		Labels:          t.Labels,
	}
	if t.Action != nil {
		rawAction, err := json.Marshal(t.Action)
		if err != nil {
			return nil, err
		}
		name := t.Action.RegisteredName()
		raw.Action[name] = rawAction
	}
	if t.Run != nil {
		rawRun, err := json.Marshal(t.Run)
		if err != nil {
			return nil, err
		}
		name := t.Run.RegisteredName()
		raw.Run[name] = rawRun
	}
	return json.Marshal(raw)
}

// HasCron return true if tasks has a cron or an every planified
func (t *Task) HasCron() bool {
	if t.Every > 0 || t.Cron != "" {
		return true
	}

	return false
}

// PrepareRechedule is used to modify start date in the future in case of a configured cron or every
// ! This does no check if cron or every is a valid value
func (t *Task) PrepareReschedule() {
	if t.Every > 0 {
		t.Start = time.Now().Add(t.Every)
	}

	if t.Cron != "" {
		sched, err := Parser.Parse(t.Cron)
		if err == nil {
			t.Start = sched.Next(time.Now())
		} else {
			t.Status = status.Error
			log.Error(fmt.Errorf("cron value %v for task %v is invalid", t.Cron, t.Id))
		}
	}

}

// InjectPredefinedEnv is used to inject or modifiy Density predefined env variables
func (t *Task) InjectPredefinedEnv() {

	now := time.Now()

	if t.Environments == nil {
		t.Environments = make(map[string]string)
	}
	t.Environments["DENSITY_STARTED_AT_DATE"] = now.Format("2006/01/02")
	t.Environments["DENSITY_STARTED_AT_TIME"] = now.Format("11:49:02")
	t.Environments["DENSITY_TASK_ID"] = t.Id.String()
	t.Environments["DENSITY"] = "true"
	t.Environments["DENSITY_RUNNER"] = t.Action.RegisteredName()
	t.Environments["DENSITY_MAX_EXECUTION_TIME"] = t.MaxExectionTime.String()

}

// NewTask init a new task
func NewTask(o string, a action.Action) Task {
	t := New()
	t.Owner = o
	t.Action = a
	return *t
}

func New() *Task {
	return &Task{
		CPU:    1,
		RAM:    1,
		Status: status.Waiting,
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
