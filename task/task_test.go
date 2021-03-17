package task

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJson(t *testing.T) {

	task := &Task{
		Owner:           "test",
		Start:           time.Now(),
		MaxExectionTime: 30 * time.Second,
		Action: &DummyAction{
			Name: "Action A",
			Wait: 10,
		},
		CPU: 2,
		RAM: 256,
	}
	raw, err := json.Marshal(task)
	assert.NoError(t, err)
	fmt.Println(string(raw))
	task2 := Task{}
	err = json.Unmarshal(raw, &task2)
	assert.NoError(t, err)
	action := task2.Action.(*DummyAction)
	assert.Equal(t, "Action A", action.Name)
}

func TestValidateLabel(t *testing.T) {

	tests := []struct {
		name   string
		label  string
		expect bool
	}{
		{
			name:   "valid",
			label:  "valid",
			expect: true,
		},
		{
			name:   "valid with hyphen",
			label:  "valid-with-hyphen",
			expect: true,
		},
		{
			name:   "valid with periods",
			label:  "valid.with.123",
			expect: true,
		},
		{
			name:   "invalid with two consecutive periods",
			label:  "invalid..",
			expect: false,
		},
		{
			name:   "invalid with unauthorized char",
			label:  "invalid)",
			expect: false,
		},
		{
			name:   "invalid combo",
			label:  "test-.",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, IsLabelValid(tt.label))
		})
	}

}
