package task

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDummyJson(t *testing.T) {
	d := &DummyAction{
		Name: "bob",
		Wg:   NewWaiter(),
	}
	d.Wg.Add(1)
	raw, err := json.Marshal(d)
	assert.NoError(t, err)
	var a DummyAction
	err = json.Unmarshal(raw, &a)
	assert.NoError(t, err)
	a.Wg.Done()
	a.Wg.Wait()
}

func TestWaiter(t *testing.T) {
	w := NewWaiter()
	assert.Equal(t, 0, w.cpt)
	w.Add(10)
	assert.Equal(t, 10, w.cpt)
	for i := 0; i < 10; i++ {
		go func() {
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			w.Done()
		}()
	}
	w.Wait()
	assert.Equal(t, 0, w.cpt)
}
