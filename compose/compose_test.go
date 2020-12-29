package compose

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

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
