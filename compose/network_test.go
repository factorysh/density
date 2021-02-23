package compose

import (
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
