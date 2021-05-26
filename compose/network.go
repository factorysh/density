package compose

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

// Subnet is a class B somewhere between 172.18.0.0 and 172.31.255.255 with a /24
//
type Subnet [2]byte

func (s Subnet) Subnet() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(172, s[0], s[1], 0),
		Mask: net.CIDRMask(24, 32),
	}
}

func (s Subnet) Next() (Subnet, error) {
	r := Subnet{}
	if s[1] < 255 {
		r[1] = s[1] + 1
		r[0] = s[0]
	} else {
		if s[0] > 31 {
			return r, fmt.Errorf("Subnet max is 172.31.255.255, %d > 31", s[0])
		}
		r[1] = 0
		r[0] = s[0] + 1
	}
	return r, nil
}

func (s Subnet) String() string {
	return fmt.Sprintf("172.%d.%d.0/24", s[0], s[1])
}

// Value of the two middles bytes
func (s Subnet) Value() uint16 {
	return uint16(s[0])*256 + uint16(s[1])
}

func ParseSubnet(txt string) (Subnet, error) {
	ip, ipnet, err := net.ParseCIDR(txt)
	if err != nil {
		return Subnet{}, err
	}
	ip = ip.To4()
	if ip == nil {
		return Subnet{}, fmt.Errorf("only ipv4 is handled : [%s]", txt)
	}
	ones, _ := ipnet.Mask.Size()
	if ones != 24 {
		return Subnet{}, fmt.Errorf("only /24 is handled : [%s] %d", txt, ones)
	}
	if ip[0] != 172 {
		return Subnet{}, fmt.Errorf("only 172.x.x.x is handled : [%s]", txt)
	}
	return Subnet{ip[1], ip[2]}, nil
}

type BySubnet []Subnet

func (b BySubnet) Len() int      { return len(b) }
func (b BySubnet) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b BySubnet) Less(i, j int) bool {
	if b[i][0] != b[j][0] {
		return b[i][0] < b[j][0]
	}
	return b[i][1] < b[j][1]
}

func (b BySubnet) next() (Subnet, error) {
	// BySubnet is sorted
	first := Subnet{18, 0}
	if len(b) == 0 {
		return first, nil
	}
	n := uint16(18 * 256)
	for i, subnet := range b { // looking for a hole
		if subnet.Value() != n {
			return b[i-1].Next() // FIXME this code is ugly, i can be 0
		}
		n++
	}
	return b[len(b)-1].Next()
}

// Add a a new Subnet, filling a hole, or a fres one
func (b BySubnet) Add() (BySubnet, error) {
	n, err := b.next()
	if err == nil {
		b = append(b, n)
	}
	return b, err
}

func (b BySubnet) Find(subnet Subnet) int {
	for i, s := range b {
		if s == subnet {
			return i
		}
	}
	return -1
}

func SubnetFromDocker(docker *client.Client) (BySubnet, error) {
	args := filters.NewArgs()
	args.Add("driver", "bridge")
	networks, err := docker.NetworkList(context.TODO(), types.NetworkListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}
	subnets := make(BySubnet, 0)
	for _, network := range networks {
		if network.Name == "bridge" { // It's the default bridge network
			continue
		}
		for _, config := range network.IPAM.Config {
			if !strings.HasPrefix(config.Subnet, "172.") {
				continue
			}
			subnet, err := ParseSubnet(config.Subnet)
			if err != nil {
				// Do I need to handle strange subnet? hum, no
				return nil, err
			}
			subnets = append(subnets, subnet)
		}
	}
	sort.Sort(subnets)

	return subnets, nil
}

type Networks struct {
	docker  *client.Client
	subnets BySubnet
	lock    *sync.Mutex
}

func NewNetworks(docker *client.Client) (*Networks, error) {
	n := &Networks{
		docker: docker,
		lock:   &sync.Mutex{},
	}
	var err error
	n.subnets, err = SubnetFromDocker(docker)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Networks) New(project string) (string, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	var err error
	for {
		n.subnets, err = n.subnets.Add()
		if err != nil {
			return "", err
		}
		last := n.subnets[len(n.subnets)-1]
		sort.Sort(n.subnets)
		networkName := fmt.Sprintf("batch-%s-%d-%d", project, last[0], last[1])

		_, err := n.docker.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
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
						Subnet: last.String(),
					},
				},
			},
		})
		if err == nil {
			return networkName, nil
		}
		if err.Error() != "Error response from daemon: Pool overlaps with other one on this address space" {
			return "", err
		} else {
			pruned, err := n.docker.NetworksPrune(context.TODO(), filters.NewArgs())
			if err != nil {
				return "", err
			}
			log.WithField("pruned", pruned.NetworksDeleted).WithField("subnet", last.String()).Warn("Network overlap, looking if next subnet is ok")
		}
	}
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
		return fmt.Errorf("One network should be found, not %d", len(networks))
	}
	subnet, err := ParseSubnet(networks[0].IPAM.Config[0].Subnet)
	if err != nil {
		return err
	}
	i := n.subnets.Find(subnet)
	if i == -1 {
		return fmt.Errorf("Not in cache : %s", network)
	}
	if i == len(n.subnets)-1 {
		n.subnets = n.subnets[:i]
	} else {
		n.subnets = append(n.subnets[:i], n.subnets[i+1:]...)
	}
	err = n.docker.NetworkRemove(context.TODO(), network)
	if err != nil {
		return err
	}
	return nil
}
