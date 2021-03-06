package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestValidator(t *testing.T) {
	cfg := `
validators:
  dummy:
`
	var v Validator
	err := yaml.Unmarshal([]byte(cfg), &v)
	assert.NoError(t, err)
	err = v.Register()
	assert.NoError(t, err)

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

	errs := v.ValidateTask(task)
	assert.Len(t, errs, 0)
}
