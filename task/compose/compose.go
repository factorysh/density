package compose

import (
	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
)

func init() {
	task.ActionValidatorRegistry["compose"] = ComposeActionValidatorFactory
}

func ComposeActionValidatorFactory(cfg map[string]interface{}) (task.ActionValidator, error) {
	cv, err := compose.NewComposeValidtor(cfg)
	if err != nil {
		return nil, err
	}
	return &ComposeActionValidator{cv}, nil
}

type ComposeActionValidator struct {
	*compose.ComposeValidator
}

func (cv *ComposeActionValidator) ValidateAction(a task.Action) []error {
	c, ok := a.(*compose.Compose)
	if !ok {
		// FIXME nil or error?
		return nil
	}
	return cv.Validate(c)
}
