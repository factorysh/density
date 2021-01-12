package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	_task "github.com/factorysh/batch-scheduler/task"
	"github.com/stretchr/testify/assert"
)

func TestRunner(t *testing.T) {
	f, err := ioutil.TempDir(os.TempDir(), "runner-")
	assert.NoError(t, err)
	fmt.Println(f)
	//defer os.Remove(f.Name())
	runner := New(f)
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
	cmd := exec.Command("docker-compose", "ps")
	cmd.Dir = f
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	assert.NoError(t, err)
	assert.True(t, false)
}
