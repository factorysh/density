package task

import (
	"fmt"

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

type Recomposator struct {
	Recomposators   map[string]map[string]interface{} `yaml:"recomposators"`
	myRecomposators map[string]ActionRecomposator
}

func (r *Recomposator) Register(docker *client.Client, projet string) error {
	r.myRecomposators = make(map[string]ActionRecomposator)
	for k, v := range r.Recomposators {
		recomposator, ok := ActionRecomposatorRegistry[k]
		if !ok {
			return fmt.Errorf("No config recomposator for %s", k)
		}
		var err error
		r.myRecomposators[k], err = recomposator(docker, projet, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Recomposator) RecomposeAction(a Action) (Action, error) {
	c, ok := r.myRecomposators[a.RegisteredName()]
	if !ok {
		return nil, fmt.Errorf("Unknow recompositor name : %s", a.RegisteredName())
	}
	return c.RecomposeAction(a)
}
