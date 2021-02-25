package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	_ "github.com/factorysh/density/compose" // register compose.Compose as task.Action
	_task "github.com/factorysh/density/task"
	_status "github.com/factorysh/density/task/status"
	"github.com/stretchr/testify/assert"
)

func TestRunner(t *testing.T) {
	f, err := ioutil.TempDir(os.TempDir(), "runner-")
	assert.NoError(t, err)
	fmt.Println(f)
	defer os.Remove(f)
	runner := New(f)
	jtask := `
	{
		"cpu": 1,
		"ram": 256,
		"action": {
			"compose": {
				"version":"3",
				"services": {
					"hello": {
						"image": "busybox",
						"environment": {
							"NAME": "$NAME"
						},
						"command": "echo $NAME"
					}
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
	assert.Equal(t, _status.Done, status)
	cmd := exec.Command("docker-compose", "logs", "--no-color", "--tail=1", "hello")
	cmd.Dir = path.Join(f, task.Id.String())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	fmt.Println("error", stderr.String())
	assert.NoError(t, err)
	logs := stdout.String()
	fmt.Println("logs:", logs)
	assert.True(t, strings.HasSuffix(logs, "Bob\n"))
}
