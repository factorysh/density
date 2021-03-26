package todo

import (
	"sync"
	"testing"
	"time"
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
	todo.Ping()
	todo.Ping()
	todo.Ping()
	time.Sleep(20 * time.Millisecond)
	todo.Ping()
	wg.Wait()
}
