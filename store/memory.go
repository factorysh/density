package store

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

func (m *MemoryStore) Delete(key []byte) error {
	delete(m.kv, string(key))
	return nil
}
