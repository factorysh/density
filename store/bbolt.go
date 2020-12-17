package store

import (
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// DefaultBucket is used as a default bucket for bolt
var DefaultBucket = []byte("default")

// BoltStore wraps all the bbol storage logic
type BoltStore struct {
	Db *bolt.DB
}

// NewBoltStore inits a BoltStore struct
func NewBoltStore(path string) (*BoltStore, error) {
	// default timeout is set to 1 sec
	db, err := bolt.Open(path, 0660, &bolt.Options{
		Timeout: 1 * time.Second,
	})
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
	if err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
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

func (bs *BoltStore) Length() int {
	var l int
	bs.Db.View(func(tx *bolt.Tx) error {
		l = tx.Bucket(DefaultBucket).Stats().KeyN
		return nil
	})
	return l
}

func (bs *BoltStore) ForEach(fn func(k, v []byte) error) error {
	bs.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)
		return b.ForEach(fn)
	})
	return nil
}
