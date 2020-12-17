package store

import "sync"

type MemoryStore struct {
	kv   map[string][]byte
	lock *sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		kv:   make(map[string][]byte),
		lock: &sync.RWMutex{},
	}
}

func (m *MemoryStore) Get(key []byte) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	v, ok := m.kv[string(key)]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (m *MemoryStore) Put(key []byte, value []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.kv[string(key)] = value
	return nil
}

func (m *MemoryStore) Delete(key []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.kv, string(key))
	return nil
}

func (m *MemoryStore) Length() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.kv)
}

func (m *MemoryStore) ForEach(fn func(k, v []byte) error) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for k, v := range m.kv {
		err := fn([]byte(k), v)
		if err != nil {
			return err
		}
	}
	return nil
}
