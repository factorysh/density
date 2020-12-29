package task

import (
	"context"
	"encoding/json"
	"testing"

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
	assert.Equal(t, Done, status)
}
