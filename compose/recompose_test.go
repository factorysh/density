package compose

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRecompose(t *testing.T) {
	docker, err := client.NewEnvClient()
	assert.NoError(t, err)
	// flush the network before starting the test
	_, err = docker.NetworksPrune(context.TODO(), filters.Args{})
	assert.NoError(t, err)
	c := NewCompose()
	err = yaml.Unmarshal([]byte(`
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
x-batch:
  key: value
`), c)
	assert.NoError(t, err)
	composator, err := NewRecomposator(docker)
	assert.NoError(t, err)
	prod, err := composator.Recompose("bob", c)
	assert.NoError(t, err)
	out, err := yaml.Marshal(prod)
	assert.NoError(t, err)
	fmt.Println(string(out))
}
