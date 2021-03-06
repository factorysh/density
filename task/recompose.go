package task

import (
	"github.com/docker/docker/client"
)

var ActionRecomposatorRegistry map[string]func(docker *client.Client, project string, cfg map[string]interface{}) (ActionRecomposator, error)

func init() {
	if ActionRecomposatorRegistry == nil {
		ActionRecomposatorRegistry = make(map[string]func(*client.Client, string, map[string]interface{}) (ActionRecomposator, error))
	}
}

type ActionRecomposator interface {
	RecomposeAction(a Action) (Action, error)
}
