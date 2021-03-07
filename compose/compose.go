package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	_run "github.com/factorysh/density/task/run"
	"gopkg.in/yaml.v3"
)

// Compose is a docker-compose project
// FIXME there is more first level keys, like volume or networks
type Compose struct {
	Networks map[string]interface{} `json:"networks,omitempty"`
	Volumes  map[string]interface{} `json:"volumes,omitempty"`
	Version  string                 `json:"version"` // Compose version
	Services map[string]interface{} `json:"services"`
	X        map[string]interface{} `json:"X,omitempty"` // The x-stuff on top level, just for aliasing
}

// NewCompose inits a compose struct
func NewCompose() *Compose {
	return &Compose{
		Services: make(map[string]interface{}),
		X:        make(map[string]interface{}),
	}

}

func (c *Compose) RegisteredName() string {
	return "compose"
}

// UnmarshalYAML is used to unmarshal a docker-compose (yaml) file
func (c *Compose) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.DocumentNode {
		return nil
	}

	for i := 0; i < len(value.Content); i += 2 {
		k := value.Content[i]
		v := value.Content[i+1]

		switch {
		case k.Value == "version":
			v.Decode(&c.Version)
		case k.Value == "services":
			var services map[string]interface{}
			err := v.Decode(&services)
			if err != nil {
				return err
			}

			for _, key := range reflect.ValueOf(services).MapKeys() {
				service, ok := services[key.String()]
				if !ok {
					return fmt.Errorf("Error while parsing service %s", key)
				}
				c.Services[key.String()] = service
			}
		case strings.HasPrefix(k.Value, "x-"):
			var xs map[string]interface{}
			err := v.Decode(&xs)
			if err != nil {
				return err
			}

			c.X[k.Value] = xs

		}
	}

	return nil
}

func (c *Compose) WalkServices(fn func(name string, value map[string]interface{}) error) error {
	for k, v := range c.Services {
		if !strings.HasPrefix(k, "x-") {
			vv, ok := v.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Not a map[string]inetrface{} %v", v)
			}
			err := fn(k, vv)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// MarshalYAML is used to marshal a Compose back to its yaml form
func (c Compose) MarshalYAML() (interface{}, error) {

	acc := map[string]interface{}{
		"version":  c.Version,
		"services": c.Services,
	}

	for k, v := range c.X {
		acc[k] = v
	}

	return acc, nil
}

// Validate compose content
func (c Compose) Validate() error {
	err := lazyEnsureBin()
	if err != nil {
		return err
	}
	p := path.Join(os.TempDir(), "validator")
	os.MkdirAll(p, 0750)
	file, err := ioutil.TempFile(p, "validate-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = yaml.NewEncoder(file).Encode(c)
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

func (c Compose) guessMainContainer() (string, error) {
	if len(c.Services) == 0 {
		return "", fmt.Errorf("'services' is not a an empty map : %p", &c.Services)
	}
	if len(c.Services) == 1 { // Easy, there is only one service
		for k := range c.Services {
			return k, nil
		}
	}
	//TODO build a DAG with depends_on, or watch for an annotation
	return "", errors.New("Multiple services handling is not yet implemented")
}

// Up compose action
func (c Compose) Up(workingDirectory string, environments map[string]string) (_run.Run, error) {
	err := lazyEnsureBin()
	if err != nil {
		return nil, err
	}
	main, err := c.guessMainContainer()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path.Join(workingDirectory, "docker-compose.yml"),
		os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return nil, err
	}
	err = yaml.NewEncoder(f).Encode(c)
	if err != nil {
		return nil, err
	}
	f.Close()

	f, err = os.OpenFile(path.Join(workingDirectory, ".env"),
		os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return nil, err
	}
	for k, v := range environments {
		// TODO escape value
		_, err = fmt.Fprintf(f, "%s=%s\n", k, v)
		if err != nil {
			return nil, err
		}
	}
	f.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "up", "--remove-orphans", "--detach")
	cmd.Dir = workingDirectory
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())
	if err != nil {
		return nil, err
	}
	fmt.Println(cmd.ProcessState.ExitCode())

	// FIXME, use docker API, not the cli
	dir := strings.Split(workingDirectory, "/")

	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(context.Background(),
		types.ContainerListOptions{
			Filters: filters.NewArgs(
				filters.KeyValuePair{
					Key:   "label",
					Value: fmt.Sprintf("com.docker.compose.service=%s", main),
				},
				filters.KeyValuePair{
					Key:   "label",
					Value: fmt.Sprintf("com.docker.compose.project=%s", dir[len(dir)-1]),
				}),
		})
	if err != nil {
		return nil, err
	}

	if len(containers) != 1 {
		return nil, fmt.Errorf("Multiple containers sharing the same service and project")
	}

	return &DockerRun{
		Path: workingDirectory,
		Id:   containers[0].ID,
	}, err
}
