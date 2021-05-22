package compose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	_run "github.com/factorysh/density/task/run"
	_status "github.com/factorysh/density/task/status"
)

func init() {
}

var _ _run.Run = &DockerRun{}

// DockerRun implements task.Run for Docker
type DockerRun struct {
	Path     string    `json:"path"`
	Id       string    `json:"id"`
	Start    time.Time `json:"start"`
	Finish   time.Time `json:"down"`
	ExitCode int       `json:"exit_code"`
	Running  bool      `json:"running"`
}

// Data returns all the data that should be exposed to the outside world
func (d *DockerRun) Data() _run.Data {
	return _run.Data{
		Start:    d.Start,
		Finish:   d.Finish,
		Runner:   d.RegisteredName(),
		ExitCode: d.ExitCode,
		Running:  d.Running,
	}
}

func (d *DockerRun) RegisteredName() string {
	return "compose"
}

func (d *DockerRun) Status() (_run.Status, int, error) {
	cli, err := client.NewEnvClient() // FIXME use a singleton
	if err != nil {
		return _run.Unkown, 0, err
	}

	// check if container exists
	ct, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{All: true, Filters: filters.NewArgs(
		filters.KeyValuePair{
			Key:   "id",
			Value: d.Id,
		},
	)})
	if err != nil {
		return _run.Unkown, 0, err
	}

	// if not early return
	if len(ct) == 0 {
		return _run.Unkown, 0, nil
	}

	// exit code is only accessible on inspect
	inspect, err := cli.ContainerInspect(context.TODO(), d.Id)
	if err != nil {
		return _run.Unkown, 0, err
	}

	var status _run.Status

	switch inspect.State.Status {
	case "created", "running", "restarting":
		status = _run.Running
	case "paused":
		status = _run.Paused
	case "removing", "exited":
		status = _run.Exited
	case "dead":
		status = _run.Dead
	default:
		status = _run.Unkown
	}

	return status, inspect.State.ExitCode, nil
}

// ID will return the Docker container ID of the main container for this run
func (d *DockerRun) ID() (string, error) {
	if d.Id == "" {
		return "", fmt.Errorf("No ID found for this run")
	}

	return d.Id, nil
}

func (d *DockerRun) Down() error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker-compose", "down", "--remove-orphans")
	cmd.Dir = d.Path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())
	d.Running = false
	return err
}

func (d *DockerRun) Wait(ctx context.Context) (_status.Status, error) {
	cli, err := client.NewEnvClient() // FIXME use a singleton
	if err != nil {
		return _status.Error, err
	}
	ctxWait, cancel := context.WithCancel(context.TODO())
	defer cancel()
	waitC, errC := cli.ContainerWait(ctxWait, d.Id, "")

	loop := true
	var status _status.Status
	for loop {
		select {
		case <-ctx.Done(): // timeout
			cancel() // don't wait anymore
			status = _status.Canceled
			loop = false
		case <-waitC: // FIXME exitcode is get later
			loop = false
		case err := <-errC:
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					loop = false
					status = _status.Timeout
				case context.Canceled:
					loop = false
					status = _status.Canceled
				default:
					return _status.Error, err
				}
			}
		}
	}
	d.Running = false
	d.Finish = time.Now()
	if status != 0 {
		// FIXME `docker-compose down`
		err = cli.ContainerKill(context.TODO(), d.Id, "KILL")
		if err != nil {
			return _status.Error, err
		}
	}
	inspect, err := cli.ContainerInspect(context.TODO(), d.Id)
	if err != nil {
		return _status.Error, err
	}
	status = _status.Error
	if inspect.State.Status == "exited" {
		if inspect.State.ExitCode == 0 {
			status = _status.Done
		}
	}
	// FIXME remove old container after waiting a bit
	d.ExitCode = inspect.State.ExitCode
	return status, nil
}
