package compose

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type Recomposator struct {
	docker  *client.Client
	lastNet *net.IPNet
	lock    *sync.Mutex
}

func NewRecomposator(docker *client.Client) *Recomposator {
	return &Recomposator{
		docker: docker,
		lastNet: &net.IPNet{
			IP:   net.ParseIP("172.16.0.0"),
			Mask: net.CIDRMask(22, 32),
		},
		lock: &sync.Mutex{},
	}
}

func (r *Recomposator) newSubnet() (*net.IPNet, bool) {
	networks, err := r.docker.NetworkList(context.TODO(), types.NetworkListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	fmt.Println(networks)
	for _, network := range networks {
		for _, config := range network.IPAM.Config {
			fmt.Println(config.Subnet)
		}
	}
	// FIXME yeah, it's ugly
	r.lock.Lock()
	defer r.lock.Unlock()
	n, ok := cidr.NextSubnet(r.lastNet, 10)
	return n, ok
}

// Recompose take a naive and validated Compose and return a Compose as it will be run
func (r *Recomposator) Recompose(name string, c *Compose) (*Compose, error) {
	networkID := uuid.New()
	n, ok := r.newSubnet()
	if !ok {
		return nil, fmt.Errorf("Can't create a new subnet %v", r.lastNet)
	}
	networkName := fmt.Sprintf("batch-%s-%s", name, networkID.String())
	_, err := r.docker.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
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
					Subnet:  n.IP.String(),
					Gateway: n.IP.String(),
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
