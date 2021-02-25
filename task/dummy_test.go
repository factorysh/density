package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_status "github.com/factorysh/density/task/status"
	"github.com/stretchr/testify/assert"
)

func TestDummyJson(t *testing.T) {
	d := &DummyAction{
		Name: "bob",
	}
	raw, err := json.Marshal(d)
	assert.NoError(t, err)
	var a DummyAction
	err = json.Unmarshal(raw, &a)
	assert.NoError(t, err)
}

func TestDummy(t *testing.T) {
	d := &DummyAction{
		Name: "bob",
	}
	run, err := d.Up("/tmp", nil)
	assert.NoError(t, err)
	ctx := context.TODO()
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, _status.Done, status)
}

func TestDummyCancel(t *testing.T) {
	d := &DummyAction{
		Name: "bob",
		Wait: 30,
	}
	run, err := d.Up("/tmp", nil)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	fmt.Println(status.String())
	assert.Equal(t, _status.Canceled, status)
}

func TestDummyTimeout(t *testing.T) {
	d := &DummyAction{
		Name: "bob",
		Wait: 30,
	}
	run, err := d.Up("/tmp", nil)
	assert.NoError(t, err)
	ctx, _ := context.WithTimeout(context.TODO(), 10*time.Millisecond)
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	fmt.Println(status.String())
	assert.Equal(t, _status.Timeout, status)
}
