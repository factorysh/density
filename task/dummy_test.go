package task

import (
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
