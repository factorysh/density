package compose

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
)

func TestNetworkNew(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	docker, err := client.NewEnvClient()
	assert.NoError(t, err)
	networks := NewNetworks(docker)
	n, err := networks.New("bob")
	assert.NoError(t, err)
	fmt.Println(n)
	err = networks.Remove(n)
	assert.NoError(t, err)
}
