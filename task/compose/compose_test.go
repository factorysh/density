package compose

import (
	"strings"
	"testing"
	"time"

	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const simpleValidator = `
---
validators:
  compose:
    NotAsDeep: 8
    NoBuild:
`

func TestCompose(t *testing.T) {
	_, ok := task.TaskValidatorRegistry["compose"]
	assert.True(t, ok)
	var v task.Validator
	err := yaml.Unmarshal([]byte(simpleValidator), &v)
	assert.NoError(t, err)
	err = v.Register()
	assert.NoError(t, err)

	for _, a := range []struct {
		tazk *task.Task
		err  string
	}{
		{
			&task.Task{
				Owner:           "test",
				Start:           time.Now(),
				MaxExectionTime: 30 * time.Second,
				Action: &compose.Compose{
					Version: "3.6",
					Services: map[string]interface{}{
						"hello": map[string]interface{}{
							"image":   "busybox",
							"command": `echo "Hello world"`,
						},
					},
				},
				CPU: 2,
				RAM: 256,
			},
			"",
		},
		{
			&task.Task{
				Owner:           "test",
				Start:           time.Now(),
				MaxExectionTime: 30 * time.Second,
				Action: &compose.Compose{
					Version: "3.6",
					Services: map[string]interface{}{
						"hello": map[string]interface{}{
							"build":   ".",
							"image":   "busybox",
							"command": `echo "Hello world"`,
						},
					},
				},
				CPU: 2,
				RAM: 256,
			},
			"Do not build inplace",
		},
	} {
		errs := v.ValidateTask(a.tazk)
		if a.err == "" {
			assert.Len(t, errs, 0)
		} else {
			assert.Len(t, errs, 1)
			assert.True(t, strings.HasPrefix(errs[0].Error(), a.err))
		}
	}
}
