package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	_task "github.com/factorysh/batch-scheduler/task"
	_status "github.com/factorysh/batch-scheduler/task/status"
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
const withVolumes = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
    volumes:
      - ./some/path/on/the/host:/some/path/inside/the/container
x-batch:
  key: value
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

x-batch:
  key: value
`

const withALotOfDeps = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
    depends_on:
      - dep
      - another

  dep:
    image: "busybox:latest"
    command: "echo dep"
    depends_on:
      - last

  another:
    image: "buxybox:latest"
    command: "echo another"

  last:
    image: "busybox:latest"
    command: "echo last"

x-batch:
  key: value

`

const withAmbiguousDeps = `
version: "3.6"
services:
  redis:
    image: redis
  pg:
    image: pg
  sidekiq:
    image: sidekiq
    depends_on:
      - redis
      - pg
  rails:
    image: rails
    depends_on:
      - redis
      - pg

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
	assert.Equal(t, _status.Done, status)
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
	assert.Equal(t, _status.Timeout, status)
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
	assert.Equal(t, _status.Canceled, status)
}

func TestNewServiceGraph(t *testing.T) {
	c := NewCompose()
	err := yaml.Unmarshal([]byte(withDependsCompose), &c)
	assert.NoError(t, err)
	graph := c.NewServiceGraph()
	assert.NoError(t, err)
	deps, ok := graph["hello"]
	assert.True(t, ok)
	assert.Equal(t, deps, []string{"dep"})
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
	assert.Equal(t, "value", xv)
}

func TestByServiceDepth(t *testing.T) {
	c := NewCompose()
	err := yaml.Unmarshal([]byte(withDependsCompose), &c)
	assert.NoError(t, err)
	graph := c.NewServiceGraph()
	depths := graph.ByServiceDepth()
	depth, ok := depths["hello"]
	assert.True(t, ok)
	assert.Equal(t, depth, 1)

	cc := NewCompose()
	err = yaml.Unmarshal([]byte(withALotOfDeps), &cc)
	assert.NoError(t, err)
	graph = cc.NewServiceGraph()
	depths = graph.ByServiceDepth()
	depth, ok = depths["hello"]
	assert.True(t, ok)
	assert.Equal(t, depth, 2)
}

func TestFindMain(t *testing.T) {
	cc := NewCompose()
	err := yaml.Unmarshal([]byte(withALotOfDeps), &cc)
	assert.NoError(t, err)
	graph := cc.NewServiceGraph()
	depths := graph.ByServiceDepth()
	main, err := depths.findLeader()
	assert.Equal(t, "hello", main)
}

func TestUnfindableMain(t *testing.T) {
	cc := NewCompose()
	err := yaml.Unmarshal([]byte(withAmbiguousDeps), &cc)
	assert.NoError(t, err)
	graph := cc.NewServiceGraph()
	depths := graph.ByServiceDepth()
	_, err = depths.findLeader()
	assert.EqualError(t, err, "Leader ambiguity between nodes rails and sidekiq")
}

func TestGetVolumesForService(t *testing.T) {
	tests := map[string]struct {
		serviceName string
		input       string
		err         error
		expected    []Volume
	}{
		"valid": {
			serviceName: "hello",
			input:       withVolumes,
			err:         nil,
			expected: []Volume{
				{
					service:       "hello",
					hostPath:      "./volumes/some/path/on/the/host",
					containerPath: "/some/path/inside/the/container",
				},
			},
		},
		"no service": {
			serviceName: "nop",
			input:       withVolumes,
			err:         fmt.Errorf("No service with name nop found"),
			expected:    nil,
		},
		"service, no volume": {
			serviceName: "hello",
			input:       validCompose,
			err:         nil,
			expected:    nil,
		},
	}

	for tname, test := range tests {
		t.Run(tname, func(t *testing.T) {
			cc := NewCompose()
			err := yaml.Unmarshal([]byte(test.input), &cc)
			assert.NoError(t, err)
			vols, err := cc.getVolumesForService(test.serviceName)

			if test.err != nil {
				assert.Errorf(t, test.err, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, vols)
			}

		})
	}
}

func TestSanitizeVolumes(t *testing.T) {

	cc := NewCompose()
	err := yaml.Unmarshal([]byte(withVolumes), &cc)
	assert.NoError(t, err)
	err = cc.SanitizeVolumes()
	assert.NoError(t, err)

	for _, srv := range cc.Services {
		service, ok := srv.(map[string]interface{})
		assert.True(t, ok)
		vols, has := service["volumes"]
		assert.True(t, has)
		volumes, ok := vols.([]interface{})
		assert.True(t, ok)
		for _, vol := range volumes {
			volume, ok := vol.(string)
			assert.True(t, ok)
			assert.True(t, strings.HasPrefix(volume, "./volumes/some/path/on/the/host"))
		}
	}
}

func TestCheckRules(t *testing.T) {
	tests := map[string]struct {
		volume Volume
		err    error
	}{
		"root": {
			volume: Volume{
				hostPath:      "/root/in/not/valid",
				containerPath: "/inside/container",
				service:       "hello",
			},
			err: fmt.Errorf("Volume /root/in/not/valid:/inside/container is not a local volume"),
		},
		"with ..": {
			volume: Volume{
				hostPath:      "./some/../../path",
				containerPath: "/inside/container",
				service:       "hello",
			},
			err: fmt.Errorf("Path ./some/../../path /inside/container contains `..`"),
		},
		"max deepness": {
			volume: Volume{
				hostPath:      "./some/very/long/a/b/c/d/e/f/g/path",
				containerPath: "/inside/container",
				service:       "hello",
			},
			err: fmt.Errorf("Volume description ./some/very/long/a/b/c/d/e/f/g/path:/inside/container reach deepnees max level 10"),
		},
		"valid path": {
			volume: Volume{
				hostPath:      "./this/is/a/valid/path:/inside/container",
				containerPath: "/inside/container",
				service:       "hello",
			},
			err: nil,
		},
	}

	for tname, test := range tests {
		tt := test
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			err := tt.volume.checkVolumeRules()
			if tt.err != nil {
				assert.Errorf(t, err, tt.err.Error())

			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJson(t *testing.T) {
	var task _task.Task
	err := json.Unmarshal([]byte(`{
		"cpu": 2,
		"ram": 128,
		"max_execution_time": "30s",
		"action": {
			"compose": {
				"version": "3",
				"services": {
					"hello": {
						"image":"busybox:latest",
						"command": "echo World"
					}
				}
			}
		}
	}`), &task)
	assert.NoError(t, err)
}
