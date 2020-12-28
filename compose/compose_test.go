package compose

import (
	"fmt"
	"testing"

	"github.com/tj/assert"
	"gopkg.in/yaml.v3"
)

const validCompose = `
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
`

const invalidCompose = `
version: '3'
services:
  hello:
    command: "echo world"
`

func TestValidate(t *testing.T) {
	tests := []struct {
		input []byte
		isOk  bool
	}{
		{
			input: []byte(validCompose),
			isOk:  true,
		},
		{
			input: []byte(invalidCompose),
			isOk:  false,
		},
	}
	for _, tc := range tests {
		var c Compose
		err := yaml.Unmarshal(tc.input, &c)
		assert.NoError(t, err)
		err = c.Validate()
		if tc.isOk {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestRunCompose(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Run valid compose file",
			input: validCompose,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var c Compose
			err := yaml.Unmarshal([]byte(tc.input), &c)
			assert.NoError(t, err)

			v, err := c.Version()
			assert.NoError(t, err)
			assert.Equal(t, "3", v)

			s, err := c.Services()
			assert.NoError(t, err)
			_, ok := s["hello"]
			assert.True(t, ok)

			k, err := c.Up("/tmp", nil)
			assert.NoError(t, err)
			fmt.Println(k)
			err = c.Down(k)
			assert.NoError(t, err)
		})
	}
}
