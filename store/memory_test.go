package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	m := NewMemoryStore()
	k := []byte("name")
	err := m.Put(k, []byte("Bob"))
	assert.NoError(t, err)
	v, err := m.Get(k)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Bob"), v)
	err = m.Delete(k)
	assert.NoError(t, err)
	v, err = m.Get(k)
	assert.NoError(t, err)
	assert.Nil(t, v)
}
