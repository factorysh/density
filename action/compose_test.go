package action

import (
	"testing"

	"github.com/factorysh/batch-scheduler/config"
	"github.com/stretchr/testify/assert"
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

func TestCompose(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		err     bool
	}{
		{
			name:    "Valid",
			input:   validCompose,
			message: "",
			err:     false},
		{
			name:    "Invalid",
			input:   invalidCompose,
			message: "The Compose file is invalid because:\nService hello has neither an image nor a build context specified. At least one must be provided.\n",
			err:     true,
		},
	}
	err := config.EnsureDirs()
	assert.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewCompose(Description{DockerCompose: tc.input})
			assert.NoError(t, err)

			message, err := c.Validate()
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.message, message)
		})
	}
}
