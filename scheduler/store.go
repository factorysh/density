package scheduler

import (
	"encoding/json"
	"errors"

	"github.com/factorysh/batch-scheduler/store"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
)

// JSONStore stores task.Task
type JSONStore struct {
	store store.Store
}

// Get a Task
func (j *JSONStore) Get(id uuid.UUID) (*task.Task, error) {
	v, err := j.store.Get([]byte(id.String()))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	var t task.Task
	err = json.Unmarshal(v, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Put task.Task
func (j *JSONStore) Put(t task.Task) error {
	if t.Id == uuid.Nil {
		return errors.New("Task wihtout id")
	}
	value, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return j.store.Put([]byte(t.Id.String()), value)
}

// Delete a task
func (j *JSONStore) Delete(id uuid.UUID) error {
	return j.store.Delete([]byte(id.String()))
}
