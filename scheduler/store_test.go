package scheduler

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/factorysh/batch-scheduler/store"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "bolt-")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	b, err := store.NewBoltStore(f.Name())
	assert.NoError(t, err)
	stores := []store.Store{store.NewMemoryStore(), b}
	for _, s := range stores {
		j := JSONStore{s}
		assert.Equal(t, 0, j.Length())
		id, err := uuid.NewRandom()
		assert.NoError(t, err)
		err = j.Put(&task.Task{
			Owner: "bob",
			Id:    id,
		})
		assert.NoError(t, err)
		v, err := j.Get(id)
		assert.NoError(t, err)
		assert.Equal(t, "bob", v.Owner)
		assert.Equal(t, 1, j.Length())
		err = j.Delete(id)
		assert.NoError(t, err)
		assert.Equal(t, 0, j.Length())
		for _, owner := range []string{"alice", "bob"} {
			id, err := uuid.NewRandom()
			assert.NoError(t, err)
			err = j.Put(&task.Task{
				Owner: owner,
				Id:    id,
			})
			assert.NoError(t, err)
		}
		assert.Equal(t, 2, j.Length())
		err = j.DeleteWithClause(func(t *task.Task) bool {
			return t.Owner == "bob"
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, j.Length())
		ok := false
		err = j.ForEach(func(t *task.Task) error {
			ok = t.Owner == "alice"
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, ok)
	}
}
