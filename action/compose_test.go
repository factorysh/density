package action

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCompose(t *testing.T) {
	tests := map[string]struct {
		input      string
		shouldFail bool
	}{
		"valid":   {input: "../compose-samples/valid-echo.yml", shouldFail: false},
		"invalid": {input: "../compose-samples/invalid-echo.yml", shouldFail: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input, err := ioutil.ReadFile(tc.input)
			assert.NoError(t, err)

			c := NewCompose(input)

			err = c.Parse()
			if tc.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestRecompose(t *testing.T) {

	expected := `services:
  hello:
    command: echo world
    image: busybox:latest
version: "3"
`

	input, err := ioutil.ReadFile("../compose-samples/valid-echo.yml")
	assert.NoError(t, err)

	c := NewCompose(input)

	err = c.Parse()
	assert.NoError(t, err)

	output, err := c.Recompose()
	assert.NoError(t, err)

	assert.Equal(t, expected, output)

}
