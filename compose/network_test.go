package compose

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetwork(t *testing.T) {
	first := Subnet{18, 0}
	n, err := first.Next()
	assert.NoError(t, err)
	for i := 0; i < 300; i++ {
		n, err = n.Next()
		assert.NoError(t, err)
	}
	assert.Equal(t, Subnet{19, 45}, n)
	assert.Equal(t, "172.18.0.0/24", first.String())
	assert.Equal(t, "172.19.45.0/24", n.String())
}

func TestNetworkParse(t *testing.T) {
	s, err := ParseSubnet("172.19.12.0/24")
	assert.NoError(t, err)
	assert.Equal(t, Subnet{19, 12}, s)
	for _, p := range []string{
		"172.19.300.0/24",
		"170.19.30.0/24",
		"172.19.30.0/20",
	} {
		_, err = ParseSubnet(p)
		assert.Error(t, err)
	}
}

func TestNetworkSort(t *testing.T) {
	ips := []Subnet{
		{18, 0},
		{22, 42},
		{19, 12},
	}

	sort.Sort(BySubnet(ips))
	assert.Equal(t, []Subnet{
		{18, 0},
		{19, 12},
		{22, 42},
	}, ips)
}

func TestNetworkCreate(t *testing.T) {
	ips := BySubnet{}
	assert.Len(t, ips, 0)
	ips, err := ips.Add()
	assert.NoError(t, err)
	assert.Len(t, ips, 1)
	assert.Equal(t, Subnet{18, 0}, ips[0])
	ips, err = ips.Add()
	assert.NoError(t, err)
	assert.Len(t, ips, 2)
	fmt.Println(ips)
	assert.Equal(t, Subnet{18, 1}, ips[1])
}

func TestNetworkCreateHole(t *testing.T) {
	holes := BySubnet{
		{18, 0},
		{18, 1},
		{18, 3},
	}
	assert.Len(t, holes, 3)
	holes, err := holes.Add()
	assert.NoError(t, err)
	assert.Len(t, holes, 4)
	assert.Equal(t, Subnet{18, 2}, holes[3])
}
