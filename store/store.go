package store

// Store kv stuff
type Store interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
	Delete([]byte) error
	Length() int
	ForEach(func(k, v []byte) error) error
	DeleteWithClause(fn func(k, v []byte) bool) error
}
