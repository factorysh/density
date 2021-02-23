package compose

import (
	"fmt"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRecompose(t *testing.T) {
	c := NewCompose()
	err := yaml.Unmarshal([]byte(`
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
x-batch:
  key: value
`), c)
	assert.NoError(t, err)
	docker, err := client.NewEnvClient()
	assert.NoError(t, err)
	composator, err := NewRecomposator(docker)
	assert.NoError(t, err)
	prod, err := composator.Recompose("bob", c)
	assert.NoError(t, err)
	out, err := yaml.Marshal(prod)
	assert.NoError(t, err)
	fmt.Println(string(out))
}
