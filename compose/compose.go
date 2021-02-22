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
	"github.com/factorysh/batch-scheduler/task"
	"github.com/factorysh/batch-scheduler/task/action"
	_run "github.com/factorysh/batch-scheduler/task/run"
	"gopkg.in/yaml.v3"
)

func init() {
	task.ActionsRegistry["compose"] = func() action.Action {
		return NewCompose()
	}
}

var composeIsHere bool = false

const volumePrefix = "./volumes"

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

func lazyEnsureBin() error {
	if composeIsHere {
		return nil
	}
	err := EnsureBin()
	if err != nil {
		return err
	}
	composeIsHere = true
	return nil
}

// Volume represent a basic docker compose volume struct
type Volume struct {
	hostPath      string
	containerPath string
	service       string
}

func (v Volume) checkVolumeRules() error {
	maxDeepness := 10

	// should start with "./"
	if !strings.HasPrefix(v.hostPath, "./") {
		return fmt.Errorf("Volume %v is not a local volume", v)
	}

	// split host path on separator
	hostParts := strings.Split(v.hostPath, string(os.PathSeparator))

	// check max deepness
	if len(hostParts) > maxDeepness {
		return fmt.Errorf("Volume description %v reach deepnees max level %d", v, maxDeepness)
	}

	// inside part (parts[0]) can't contain '..'
	for _, part := range hostParts {
		if part == ".." {
			return fmt.Errorf("Path %v contains `..`", v.hostPath)
		}
	}

	return nil
}

// addPrefix needs to be idempotent, if ./volumes is present, to prepend it another time
func (v *Volume) addPrefix() {

	if !strings.HasPrefix(v.hostPath, "./volumes") {
		v.hostPath = fmt.Sprintf("./volumes/%s", strings.TrimLeft(v.hostPath, "./"))
	}
}

// toVolumeString returns the content of a volume struct to a compose volume string
func (v Volume) toVolumeString() string {
	return fmt.Sprintf("%s:%s", v.hostPath, v.containerPath)
}

// Compose is a docker-compose project
// FIXME there is more first level keys, like volume or networks
type Compose struct {
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

	err := c.SanitizeVolumes()
	if err != nil {
		return err
	}

	return nil
}

// getVolumesForService is used to retreive a list of Volume struct from a service
func (c *Compose) getVolumesForService(name string) ([]Volume, error) {

	srv, ok := c.Services[name]
	if !ok {
		return nil, fmt.Errorf("No service with name %s found", name)
	}

	service, ok := srv.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Can't cast service %s into a map", name)
	}

	vols, has := service["volumes"]
	if !has {
		return nil, nil
	}

	volumes, ok := vols.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Can't cast volumes %v into a map", vols)
	}

	// create array for all volumes
	allVolumes := make([]Volume, len(volumes))
	for i, vol := range volumes {
		volume, ok := vol.(string)
		if !ok {
			return nil, fmt.Errorf("Can't cast volume %v into a volume string", vol)
		}
		// check that volume contains two parts
		parts := strings.Split(volume, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Volume %v do not contains two parts", volume)
		}
		allVolumes[i] = Volume{
			service:       name,
			hostPath:      parts[0],
			containerPath: parts[1],
		}

	}

	return allVolumes, nil
}

// SanitizeVolumes is used to ensure that volumes are plug the right way when Up is called
func (c *Compose) SanitizeVolumes() error {
	for key, srv := range c.Services {
		volumes, err := c.getVolumesForService(key)
		if err != nil {
			return err
		}

		if volumes == nil {
			continue
		}

		service, ok := srv.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Can't cast compose structure into a map %v", c.Services)
		}

		sanitized := make([]interface{}, len(volumes))
		for i, vol := range volumes {
			err := vol.checkVolumeRules()
			if err != nil {
				return err
			}

			vol.addPrefix()

			sanitized[i] = vol.toVolumeString()
		}

		// go for sanitized array
		service["volumes"] = sanitized
		// replace sanitized array
		c.Services[key] = service
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

func (c Compose) ensureVolumesInWD(workingDir string) error {

	for key := range c.Services {
		volumes, err := c.getVolumesForService(key)
		if err != nil {
			return err
		}

		for _, volume := range volumes {
			fmt.Println(path.Join(workingDir, volume.hostPath))
			err := os.MkdirAll(path.Join(workingDir, volume.hostPath), 0755)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

	err = c.ensureVolumesInWD(workingDirectory)
	if err != nil {
		return nil, err
	}

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
