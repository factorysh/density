package action

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Compose is a docker-compose project
type Compose struct {
	path    string
	service string
	env     map[string]string
}

// Action run the project
func (c *Compose) Action(ctx context.Context) error {
	return nil
}

// EnsureBin will ensure that docker-compose is found in $PATH
func EnsureBin() error {
	var name = "docker-compose"
	var out bytes.Buffer

	cmd := exec.Command("whereis", "-b", name)
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return err
	}

	sanitized := strings.TrimRight(out.String(), "\n")
	matched, err := regexp.Match(fmt.Sprintf("%s: .*/%s", name, name), []byte(sanitized))
	if !matched {
		return errors.Errorf("Executable %s not found", name)
	}

	return nil

}
