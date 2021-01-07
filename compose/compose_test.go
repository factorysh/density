package compose

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const validCompose = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
x-batch:
  key: value
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

const withDependsCompose = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
    depends_on:
      - dep

  dep:
    image: "busybox:latest"
    command: "echo dep"
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
		c := NewCompose()
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
	c := NewCompose()
	err := yaml.Unmarshal([]byte(validCompose), &c)
	assert.NoError(t, err)

	assert.Equal(t, "3", c.Version)

	_, ok := c.Services["hello"]
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
	c := NewCompose()
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
	c := NewCompose()
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

func TestNewServiceGraph(t *testing.T) {
	c := NewCompose()
	err := yaml.Unmarshal([]byte(withDependsCompose), &c)
	assert.NoError(t, err)
	graph, err := c.NewServiceGraph()
	assert.NoError(t, err)
	fmt.Println(graph)
}

func TestUnmarshal(t *testing.T) {
	c := NewCompose()
	err := yaml.Unmarshal([]byte(withDependsCompose), c)
	assert.NoError(t, err)
	assert.Equal(t, "3", c.Version)
	hello, ok := c.Services["hello"].(map[string]interface{})
	assert.True(t, ok)
	cmd, ok := hello["command"]
	assert.True(t, ok)
	assert.Equal(t, "echo world", cmd)
	x, ok := c.X["x-batch"].(map[string]interface{})
	assert.True(t, ok)
	xv, ok := x["key"]
	assert.True(t, ok)
	assert.Equal(t, xv, "value")
}
