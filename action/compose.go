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
func NewCompose(desc []byte) (*Compose, error) {
	c := make(map[interface{}]interface{})

	err := yaml.Unmarshal(desc, c)
	if err != nil {
		return nil, err
	}

	return &Compose{
		raw:     string(desc),
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

// Content will return cotent par as []byte
func (c *Compose) Content() ([]byte, error) {
	b, err := yaml.Marshal(c.content)
	return b, err
}

// Write will write the content part of the struct into a docker-compose file in specic
func (c *Compose) Write(dir string) error {

	data, err := c.Content()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dir, "docker-compose.yml"), data, 0755)

	return err
}

// Run compose action
func (c *Compose) Run(ctx context.Context) error {
	uuid, ok := FromCtxUUID(ctx)
	if !ok {
		return errors.New("Run aborted due to missing UUID in context value")
	}

	err := ensureCtxDir(uuid)
	if err != nil {
		return err
	}

	wd := fmt.Sprintf("%s/wd/%s", config.GetDataDir(), uuid)
	err = c.Write(wd)
	if err != nil {
		return err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "up")
	cmd.Dir = wd
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())

	return err
}

// Action run the project
func (c *Compose) Action(ctx context.Context) error {
	return nil
}

// EnsureBin will ensure that docker-compose is found in $PATH
func EnsureBin() error {
	var name = "docker-compose"
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("whereis", "-b", name)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		print(stderr.String())
		return err
	}

	sanitized := strings.TrimRight(out.String(), "\n")
	matched, err := regexp.Match(fmt.Sprintf("%s: .*/%s", name, name), []byte(sanitized))
	if !matched {
		return errors.Errorf("Executable %s not found", name)
	}

	return nil

}

// ensureCtxDir ensures a working directory, per uuid
func ensureCtxDir(uuid string) error {

	b := config.GetDataDir()

	err := os.MkdirAll(fmt.Sprintf("%s/%s/%s", b, "wd", uuid), 0755)

	return err
}
