package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
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

// DefaultBucket is used as a default bucket for bolt
var DefaultBucket = []byte("default")

// BoltStore wraps all the bbol storage logic
type BoltStore struct {
	Db *bolt.DB
}

// NewBoltStore inits a BoltStore struct
func NewBoltStore(path string) (*BoltStore, error) {
	// default timeout is set to 1 sec
	db, err := bolt.Open(path, 0666, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// create a default bucket if not exists
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(DefaultBucket)
		return nil
	})

	return &BoltStore{
		Db: db,
	}, err
}

// Put value associtated to key in the datastore
func (bs *BoltStore) Put(key []byte, value []byte) error {

	err := bs.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)
		if b == nil {
			return fmt.Errorf("bucket %s does not exists", DefaultBucket)
		}

		err := b.Put(key, value)

		return err
	})

	return err
}

// Get a value using it's key
func (bs *BoltStore) Get(key []byte) ([]byte, error) {

	// a value to old data
	var value []byte

	err := bs.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)
		if b == nil {
			return fmt.Errorf("bucket %s does not exists", DefaultBucket)
		}

		v := b.Get(key)
		value = make([]byte, len(v))
		copy(value, v)

		return nil
	})

	return value, err
}

// Delete a value using it's key
func (bs *BoltStore) Delete(key []byte) error {

	err := bs.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)
		if b == nil {
			return fmt.Errorf("bucket %s does not exists", DefaultBucket)
		}

		return b.Delete(key)
	})

	return err

}
