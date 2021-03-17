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
		label  *Label
		expect error
	}{
		{
			name: "valid",
			label: &Label{
				Key:   "valid",
				Value: "valid",
			},
			expect: nil,
		},
		{
			name: "valid with hyphen",
			label: &Label{
				Key:   "valid-with-hyphen",
				Value: "valid-hyphen",
			},
			expect: nil,
		},
		{
			name: "valid with periods",
			label: &Label{
				Key:   "valid.with.123",
				Value: "valid.123",
			},
			expect: nil,
		},
		{
			name: "invalid key",
			label: &Label{
				Key:   "invalid..",
				Value: "valid.123",
			},
			expect: fmt.Errorf("Label key `invalid..` do not respect labels format policy"),
		},
		{
			name: "invalid value",
			label: &Label{
				Key:   "valid.key",
				Value: "invalid--value",
			},
			expect: fmt.Errorf("Label value `invalid--value` do not respect labels format policy"),
		},
		{
			name: "invalid combo",
			label: &Label{
				Key:   "invalid.-key",
				Value: "invalid--value",
			},
			expect: fmt.Errorf("Label key `invalid.-key` do not respect labels format policy"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.label.Validate())
		})
	}

}
