package store

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	m := NewMemoryStore()
	assert.Equal(t, 0, m.Length())
	k := []byte("name")
	err := m.Put(k, []byte("Bob"))
	assert.NoError(t, err)
	assert.Equal(t, 1, m.Length())
	v, err := m.Get(k)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Bob"), v)
	err = m.Delete(k)
	assert.NoError(t, err)
	assert.Equal(t, 0, m.Length())
	v, err = m.Get(k)
	assert.NoError(t, err)
	assert.Nil(t, v)
	for _, name := range []string{"pim", "pam", "poum"} {
		m.Put([]byte(name), []byte{})
	}
	assert.Equal(t, 3, m.Length())
	names := make([]string, 0)
	err = m.ForEach(func(k, v []byte) error {
		names = append(names, string(k))
		return nil
	})
	assert.NoError(t, err)
	sort.Strings(names)
	assert.Equal(t, []string{"pam", "pim", "poum"}, names)
}
