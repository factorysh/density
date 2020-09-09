package action

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompose(t *testing.T) {
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
