package runner

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	_task "github.com/factorysh/batch-scheduler/task"
	"github.com/stretchr/testify/assert"
)

func TestRunner(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "runner-")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	runner := New(f.Name())
	jtask := `
	{
		"cpu": 1,
		"ram": 256,
		"action": {
			"version":"3",
			"services": {
				"hello": {
					"image": "busybox",
					"command": "echo $NAME"
				}
			}
		},
		"environments": {
			"NAME": "Bob"
		}
	}
	`
	var task _task.Task
	err = json.Unmarshal([]byte(jtask), &task)
	assert.NoError(t, err)
	run, err := runner.Up(&task)
	assert.NoError(t, err)
	ctx := context.TODO()
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, _task.Done, status)
}
