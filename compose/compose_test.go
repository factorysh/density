package compose

import (
	"context"
	"testing"

	"github.com/tj/assert"
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
			c, err := FromYAML([]byte(tc.input))
			assert.NoError(t, err)

			ctx := context.Background()
			err = c.Run(ctx, "/tmp", nil)
			assert.NoError(t, err)
		})
	}
}
