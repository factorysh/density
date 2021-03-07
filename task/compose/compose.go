package compose

import (
	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
	"github.com/factorysh/density/task/action"
	"github.com/factorysh/density/task/run"
)

func init() {
	task.ActionValidatorRegistry["compose"] = ComposeActionValidatorFactory
	task.ActionsRegistry["compose"] = func() action.Action {
		return compose.NewCompose()
	}
	task.RunRegistry["compose"] = func() run.Run {
		return &compose.DockerRun{
			Id:   "",
			Path: "",
		}
	}
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
