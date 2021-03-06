package compose

import (
	"fmt"

	"github.com/docker/docker/client"
	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
)

func init() {
	task.ActionRecomposatorRegistry["compose"] = ComposeActionRecomposatorFactory
}

func ComposeActionRecomposatorFactory(docker *client.Client, projet string, cfg map[string]interface{}) (task.ActionRecomposator, error) {
	r, err := compose.NewRecomposator(docker, cfg)
	if err != nil {
		return nil, err
	}
	return &ComposeActionRecompose{
		r,
		projet,
	}, nil
}

type ComposeActionRecompose struct {
	*compose.Recomposator
	projet string
}

func (r *ComposeActionRecompose) RecomposeAction(a task.Action) (task.Action, error) {
	cmp, ok := a.(*compose.Compose)
	if !ok {
		return nil, fmt.Errorf("Not o compose: %v", a)
	}
	return r.Recompose(r.projet, cmp)
}
