package scheduler

import (
	"encoding/json"
	"errors"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
)

type JSONStore struct {
	store Store
}

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

func (j *JSONStore) Delelete(id uuid.UUID) error {
	return j.store.Delete([]byte(id.String()))
}

type Store interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
	Delete([]byte) error
}

type MemoryStore struct {
	kv map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		kv: make(map[string][]byte),
	}
}

func (m *MemoryStore) Get(key []byte) ([]byte, error) {
	v, ok := m.kv[string(key)]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (m *MemoryStore) Put(key []byte, value []byte) error {
	m.kv[string(key)] = value
	return nil
}

func (m *MemoryStore) Delelete(key []byte) error {
	delete(m.kv, string(key))
	return nil
}
