package compose

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	_network "github.com/factorysh/density/network"
	log "github.com/sirupsen/logrus"
)

func SubnetFromDocker(docker *client.Client) ([]*net.IPNet, error) {
	args := filters.NewArgs()
	args.Add("driver", "bridge")
	networks, err := docker.NetworkList(context.TODO(), types.NetworkListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}
	subnets := make([]*net.IPNet, 0)
	for _, network := range networks {
		if network.Name == "bridge" { // It's the default bridge network
			continue
		}
		for _, config := range network.IPAM.Config {
			_, subnet, err := net.ParseCIDR(config.Subnet)
			if err != nil {
				return nil, err
			}
			subnets = append(subnets, subnet)

		}
	}
	return subnets, nil
}

type Networks struct {
	docker *client.Client
	lock   *sync.Mutex
	min    *net.IPNet
	max    *net.IPNet
	mask   net.IPMask
}

func NewNetworks(docker *client.Client) *Networks {
	return &Networks{
		docker: docker,
		lock:   &sync.Mutex{},
		min: &net.IPNet{
			IP:   net.IPv4(172, 18, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		max: &net.IPNet{
			IP:   net.IPv4(172, 24, 32, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		mask: net.IPv4Mask(255, 255, 255, 0),
	}
}

func (n *Networks) New(project string) (string, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	l := log.WithField("project", project)

	// seach for an existing network for this project
	networks, err := n.docker.NetworkList(context.TODO(),
		types.NetworkListOptions{Filters: filters.NewArgs(
			filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("batch=%s", project)})})
	if err != nil {
		return "", err
	}

	// if a network exists, just reuse it
	if len(networks) != 0 {
		networkName := networks[0].Name
		l = l.WithField("reusing_network", networkName)
		l.Info()
		return networkName, nil
	}

	now := time.Now()
	subnets, err := SubnetFromDocker(n.docker)
	l = l.WithField("find_subnets", time.Since(now))
	if err != nil {
		return "", err
	}
	for i := 0; i < 2; i++ {
		now = time.Now()
		subnet, err := _network.NextAvailableNetwork(subnets, n.min, n.max, n.mask)
		l = l.WithField("find_next_network", time.Since(now)).WithField("subnet", subnet)
		networkName := fmt.Sprintf("batch-%s-%d-%d", project, subnet.IP[2], subnet.IP[3])

		now = time.Now()
		_, err = n.docker.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
			CheckDuplicate: true,
			EnableIPv6:     false,
			Scope:          "local",
			Driver:         "bridge",
			Labels: map[string]string{
				"batch": project,
			},
			Attachable: true,
			IPAM: &network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{
					{
						Subnet: subnet.String(),
					},
				},
			},
		})
		l = l.WithField("create_network", time.Since(now))
		if err == nil {
			l.Info()
			return networkName, nil
		}
		l = l.WithError(err)
		pruned, err := n.docker.NetworksPrune(context.TODO(), filters.NewArgs())
		if err != nil {
			return "", err
		}
		l.WithField("pruned", pruned.NetworksDeleted).Warn("Network overlap, looking if next subnet is ok")
	}
	l.Error()
	return "", errors.New("can't add a network")
}

func (n *Networks) Remove(network string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	args := filters.NewArgs()
	args.Add("name", network)
	networks, err := n.docker.NetworkList(context.TODO(), types.NetworkListOptions{
		Filters: args,
	})
	if err != nil {
		return err
	}
	if len(networks) != 1 {
		return fmt.Errorf("one network should be found, not %d", len(networks))
	}
	err = n.docker.NetworkRemove(context.TODO(), network)
	if err != nil {
		return err
	}
	return nil
}
