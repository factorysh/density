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
