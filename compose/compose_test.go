package compose

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/tj/assert"
	"gopkg.in/yaml.v3"
)

const validCompose = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
`

const sleepCompose = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "sleep 30"
`
const invalidCompose = `
version: '3'
services:
  hello:
    command: "echo world"
`

func TestValidate(t *testing.T) {
	tests := []struct {
		input []byte
		isOk  bool
	}{
		{
			input: []byte(validCompose),
			isOk:  true,
		},
		{
			input: []byte(invalidCompose),
			isOk:  false,
		},
	}
	for _, tc := range tests {
		var c Compose
		err := yaml.Unmarshal(tc.input, &c)
		assert.NoError(t, err)
		err = c.Validate()
		if tc.isOk {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestRunCompose(t *testing.T) {
	var c Compose
	err := yaml.Unmarshal([]byte(validCompose), &c)
	assert.NoError(t, err)

	v, err := c.Version()
	assert.NoError(t, err)
	assert.Equal(t, "3", v)

	s, err := c.Services()
	assert.NoError(t, err)
	_, ok := s["hello"]
	assert.True(t, ok)

	dir, err := ioutil.TempDir(os.TempDir(), "compose-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	run, err := c.Up(dir, nil)
	assert.NoError(t, err)
	fmt.Println(run)
	ctx := context.TODO()
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, task.Done, status)
}

func TestRunComposeTimeout(t *testing.T) {
	var c Compose
	err := yaml.Unmarshal([]byte(sleepCompose), &c)
	assert.NoError(t, err)
	dir, err := ioutil.TempDir(os.TempDir(), "compose-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	run, err := c.Up(dir, nil)
	assert.NoError(t, err)
	ctx, _ := context.WithTimeout(context.TODO(), time.Second)
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, task.Timeout, status)
}

func TestRunComposeCancel(t *testing.T) {
	var c Compose
	err := yaml.Unmarshal([]byte(sleepCompose), &c)
	assert.NoError(t, err)
	dir, err := ioutil.TempDir(os.TempDir(), "compose-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	run, err := c.Up(dir, nil)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	status, err := run.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, task.Canceled, status)
}
