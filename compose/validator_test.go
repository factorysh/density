package compose

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestValidator(t *testing.T) {
	c := NewCompose()
	for _, tc := range []struct {
		name  string
		input string
		err   string
	}{
		{
			name: "with build",
			input: `
version: "3.6"
services:
  bob:
    build: "."
`,
			err: "Do not build",
		},
		{
			name: "with logging",
			input: `
version: "3.6"
services:
  bob:
    logging:
      driver: syslog
`,
			err: "The logging is",
		},
		{
			name: "correct volume",
			input: `
version: "3.6"
services:
  bob:
    volumes:
      - ./plop:/plop
`,
			err: "",
		},
		{
			name: "absolute path",
			input: `
version: "3.6"
services:
  bob:
    volumes:
      - /etc/shadow:/plop
`,
			err: "Relative volume only",
		},
		{
			name: "with ..",
			input: `
version: "3.6"
services:
  bob:
    volumes:
      - ./../etc/shadow:/plop
`,
			err: "Path with ..",
		},
		{
			name: "too deep",
			input: `
version: "3.6"
services:
  bob:
    volumes:
      - ./a/b/c/d/e/f/g/h/i/j:/plop
`,
			err: "Path is too deep",
		},
	} {
		err := yaml.Unmarshal([]byte(tc.input), &c)
		assert.NoError(t, err, tc.input)
		errs := StandardValidtator.Validate(c)
		if tc.err == "" {
			assert.Len(t, errs, 0, tc)
		} else {
			assert.Len(t, errs, 1, tc)
			if len(errs) > 0 {
				assert.True(t, strings.HasPrefix(errs[0].Error(), tc.err),
					errs[0])
			}
		}
	}
}
