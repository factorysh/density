package compose

import (
	"context"
	"fmt"
	"testing"

	"github.com/PaesslerAG/jsonpath"
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
    volumes:
      - ./tmp:/plop:ro
x-batch:
  key: value
`), c)
	assert.NoError(t, err)
	composator, err := StandardRecomposator(docker)
	assert.NoError(t, err)
	prod, err := composator.Recompose("bob", c)
	assert.NoError(t, err)
	out, err := yaml.Marshal(prod)
	assert.NoError(t, err)
	fmt.Println(string(out))
	volumes, err := jsonpath.Get("$.hello.volumes", prod.Services)
	assert.NoError(t, err)
	assert.Equal(t, []string{"./volumes/tmp:/plop:ro"}, volumes)
}
