package runners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureBin(t *testing.T) {
	tests := map[string]struct {
		input      string
		shouldFail bool
	}{
		"valid":   {input: "docker-compose", shouldFail: false},
		"invalid": {input: "dckr-cmps", shouldFail: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := EnsureBin(tc.input)
			if tc.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}

}
