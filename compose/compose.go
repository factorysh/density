package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// EnsureBin will ensure that docker-compose is found in $PATH
func EnsureBin() error {
	var name = "docker-compose"
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("which", name)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		print(stderr.String())
		return fmt.Errorf("%s not found: %s", name, err.Error())
	}
	return nil
}

// Compose is a docker-compose project
type Compose struct {
	raw     string
	content map[string]interface{}
	tmpFile string
}

// NewCompose creates a new compose struct that implements the action.Job interface
func FromYAML(desc []byte) (*Compose, error) {
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
func (c *Compose) Validate() error {
	file, err := ioutil.TempFile(fmt.Sprintf("%s/%s", c.tmpFile, "validator"), "")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte(c.raw))
	if err != nil {
		return err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "-f", file.Name(), "config", "-q")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return errors.New(stderr.String())
	}

	return err
}

// ToYAML dump compose file as YAML
func (c *Compose) ToYAML() ([]byte, error) {
	return yaml.Marshal(c.content)
}

// Run compose action
func (c *Compose) Run(ctx context.Context, workingDirectory string, environments map[string]string) error {
	f, err := os.OpenFile(fmt.Sprintf("%s/docker-compose.yml", workingDirectory),
		os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	err = yaml.NewEncoder(f).Encode(c.content)
	if err != nil {
		return err
	}
	f.Close()

	f, err = os.OpenFile(fmt.Sprintf("%s/.env", workingDirectory),
		os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	for k, v := range environments {
		// TODO escape value
		_, err = fmt.Fprintf(f, "%s=%s\n", k, v)
		if err != nil {
			return err
		}
	}
	f.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "up", "--abort-on-container-exit")
	cmd.Dir = workingDirectory
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())

	return err
}

func (c *Compose) Version() (string, error) {
	v, ok := c.content["version"]
	if !ok {
		return "", errors.New("version is mandatory")
	}
	vv, ok := v.(string)
	if !ok {
		return "", errors.New("version must be a string")
	}
	return vv, nil
}

func (c *Compose) Services() (map[string]interface{}, error) {
	s, ok := c.content["services"]
	if !ok {
		return nil, errors.New("services is mandatory")
	}
	ss, ok := s.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Wrong format : %v", s)
	}
	return ss, nil
}
