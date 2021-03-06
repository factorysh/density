package task

import (
	"testing"

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

	errs := v.ValidateAction(&DummyAction{
		Name: "Action A",
		Wait: 10,
	})
	assert.Len(t, errs, 0)
}
