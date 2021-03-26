package todo

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTodo(t *testing.T) {
	todo := New()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			select {
			case <-todo.Wait():
				time.Sleep(10 * time.Millisecond)
				err := todo.Done()
				assert.NoError(t, err)
				wg.Done()
			case <-time.After(10 * time.Second):
				panic("Timeout")
			}
		}
	}()
	ok := todo.Ping()
	assert.True(t, ok)
	ok = todo.Ping()
	assert.False(t, ok)
	ok = todo.Ping()
	assert.False(t, ok)
	time.Sleep(20 * time.Millisecond)
	ok = todo.Ping()
	assert.True(t, ok)
	wg.Wait()
}

func TestDone(t *testing.T) {
	todo := New()
	err := todo.Done()
	assert.Error(t, err)
}

func TestFlush(t *testing.T) {
	todo := New()
	todo.Ping()
	assert.Len(t, todo.wait, 1)
	err := todo.Done()
	assert.NoError(t, err)
	assert.Len(t, todo.wait, 0)
}
