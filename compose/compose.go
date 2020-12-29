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

	"github.com/docker/docker/client"
	"github.com/factorysh/batch-scheduler/task"
	"gopkg.in/yaml.v3"
)

var composeIsHere bool = false

// DockerRun implements task.Run
type DockerRun struct {
	Path string `json:"path"`
	Id   string `json:"id"`
}

func (d *DockerRun) Down() error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = d.Path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())
	return err
}

func (d *DockerRun) Wait(ctx context.Context) (task.Status, error) {
	cli, err := client.NewEnvClient() // FIXME use a singleton
	if err != nil {
		return task.Error, err
	}
	_, errC := cli.ContainerWait(ctx, d.Id, "")
	// FIXME if timeout, kill the container
	if err := <-errC; err != nil {
		return task.Error, err
	}
	inspect, err := cli.ContainerInspect(context.TODO(), d.Id)
	if err != nil {
		return task.Error, err
	}
	status := task.Error
	if inspect.State.Status == "exited" {
		if inspect.State.ExitCode == 0 {
			status = task.Done
		}
	}
	return status, nil
}

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

// Compose is a docker-compose project
type Compose map[string]interface{}

// Validate compose content
func (c Compose) Validate() error {
	err := lazyEnsureBin()
	if err != nil {
		return err
	}
	tmpfile := os.Getenv("BATCH_TMP")
	if tmpfile == "" {
		tmpfile = "/tmp"
	}
	tmpdir, err := ioutil.TempDir(tmpfile, "")
	if err != nil {
		return err
	}
	err = os.MkdirAll(tmpdir, 0750)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path.Join(tmpdir, "validator"), os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer os.Remove(tmpdir)

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
	services, err := c.Services()
	if err != nil {
		return "", err
	}
	if len(services) == 0 {
		return "", fmt.Errorf("'services' is not a an empty map : %p", services)
	}
	if len(services) == 1 { // Easy, there is only one service
		for k := range services {
			return k, nil
		}
	}
	//TODO build a DAG with depends_on, or watch for an annotation
	return "", errors.New("Multiple services handling is not yet implemented")
}

// Up compose action
func (c Compose) Up(workingDirectory string, environments map[string]string) (task.Run, error) {
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

	// FIXME, use docker API, not the cli
	cmd = exec.Command("docker", "inspect", "--format", "{{ .Id }}", fmt.Sprintf("%s_%s_1", workingDirectory, main))
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())
	if err != nil {
		return nil, err
	}

	return &DockerRun{
		Path: workingDirectory,
		Id:   stdout.String(),
	}, err
}

// Version check if version is set in docker compose file
func (c Compose) Version() (string, error) {
	v, ok := c["version"]
	if !ok {
		return "", errors.New("version is mandatory")
	}
	vv, ok := v.(string)
	if !ok {
		return "", errors.New("version must be a string")
	}
	return vv, nil
}

// Services gets all the services from a compose file
func (c Compose) Services() (map[string]interface{}, error) {
	s, ok := c["services"]
	if !ok {
		return nil, errors.New("services is mandatory")
	}
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("Wrong format : %v", s)
	}
	r := make(map[string]interface{})
	for _, k := range v.MapKeys() {
		if k.Kind() != reflect.String {
			return nil, fmt.Errorf("Wrong key format: %v", k)
		}
		r[k.String()] = v.MapIndex(k)
	}
	return r, nil
}
