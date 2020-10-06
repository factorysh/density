package action

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/factorysh/batch-scheduler/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Compose is a docker-compose project
type Compose struct {
	raw     string
	content map[interface{}]interface{}
}

// NewCompose creates a new compose struct that implements the action.Job interface
func NewCompose(desc Description) (*Compose, error) {
	c := make(map[interface{}]interface{})

	err := yaml.Unmarshal([]byte(desc.DockerCompose), c)
	if err != nil {
		return nil, err
	}

	return &Compose{
		raw:     desc.DockerCompose,
		content: c,
	}, err
}

// Validate compose content
func (c *Compose) Validate() (string, error) {
	b := config.GetDataDir()
	file, err := ioutil.TempFile(fmt.Sprintf("%s/%s", b, "validator"), "")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte(c.raw))
	if err != nil {
		return "", err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "-f", file.Name(), "config", "-q")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return "", err

}

// Run compose action
func (c *Compose) Run(cxt context.Context) error {
	return nil
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
