package compose

import (
	"fmt"

	"github.com/docker/docker/client"
)

type Recomposator struct {
	docker   *client.Client
	networks *Networks
}

func NewRecomposator(docker *client.Client) (*Recomposator, error) {
	n, err := NewNetworks(docker)
	if err != nil {
		return nil, err
	}
	return &Recomposator{
		docker:   docker,
		networks: n,
	}, nil
}

// Recompose take a naive and validated Compose and return a Compose as it will be run
func (r *Recomposator) Recompose(name string, c *Compose) (*Compose, error) {
	networkName, err := r.networks.New(name)
	if err != nil {
		return nil, err
	}
	prod := &Compose{
		Services: copyMap(c.Services),
		Version:  c.Version,
		X:        copyMap(c.X),
		Networks: map[string]interface{}{
			"default": map[string]interface{}{
				"external": map[string]interface{}{
					"name": networkName,
				},
			},
		},
	}
	prod.WalkServices(func(name string, service map[string]interface{}) error {
		labelsRaw, ok := service["labels"]
		if !ok {
			service["labels"] = map[string]string{
				"batch": name,
			}
			return nil
		}
		labels, ok := labelsRaw.(map[string]string)
		if !ok {
			return fmt.Errorf("labels is not a map %v", labelsRaw)
		}
		labels["batch"] = name
		return nil
	})
	return prod, nil
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
