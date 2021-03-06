package compose

import (
	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
)

func init() {
	task.TaskValidatorRegistry["compose"] = ComposeTaskValidatorFactory
}

func ComposeTaskValidatorFactory(cfg map[string]interface{}) (task.TaskValidator, error) {
	cv, err := compose.NewComposeValidtor(cfg)
	if err != nil {
		return nil, err
	}
	return &ComposeTaskValidator{cv}, nil
}

type ComposeTaskValidator struct {
	*compose.ComposeValidator
}

func (cv *ComposeTaskValidator) ValidateTask(t *task.Task) []error {
	c, ok := t.Action.(*compose.Compose)
	if !ok {
		// FIXME nil or error?
		return nil
	}
	return cv.Validate(c)
}
