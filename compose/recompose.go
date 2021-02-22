package compose

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type Recomposator struct {
	docker *client.Client
	used   []uint32
	lock   *sync.Mutex
}

func NewRecomposator(docker *client.Client) *Recomposator {
	return &Recomposator{
		docker: docker,
		used:   make([]uint32, 0),
		lock:   &sync.Mutex{},
	}
}

func (r *Recomposator) newSubnet() (uint32, error) {
	// FIXME yeah, it's ugly
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(r.used) == 0 {
		r.used = []uint32{0}
		return 0, nil
	}
	n := r.used[len(r.used)-1] + 12
	if n > 254 {
		return 0, errors.New("Subnet exhausting")
	}
	r.used = append(r.used, n)
	return n, nil
}

// Recompose take a naive and validated Compose and return a Compose as it will be run
func (r *Recomposator) Recompose(name string, c *Compose) (*Compose, error) {
	networkID := uuid.New()
	n, err := r.newSubnet()
	if err != nil {
		return nil, err
	}
	networkName := fmt.Sprintf("batch-%s-%s", name, networkID.String())
	_, err = r.docker.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
		CheckDuplicate: true,
		EnableIPv6:     false,
		Scope:          "local",
		Driver:         "bridge",
		Labels: map[string]string{
			"batch": name,
		},
		Attachable: true,
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  fmt.Sprintf("172.16.%d.0/22", n),
					Gateway: fmt.Sprintf("172.16.%d.1", n),
				},
			},
		},
	})
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
