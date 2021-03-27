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
				todo.Done()
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

func TestFlush(t *testing.T) {
	todo := New()
	todo.Ping()
	assert.Len(t, todo.wait, 1)
	todo.Done()
	assert.Len(t, todo.wait, 0)
	todo.Done() // nothing happened, double Done is ok
}
