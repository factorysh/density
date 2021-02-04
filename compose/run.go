package compose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/factorysh/batch-scheduler/task"
	_run "github.com/factorysh/batch-scheduler/task/run"
	_status "github.com/factorysh/batch-scheduler/task/status"
)

func init() {
	task.RunRegistry["compose"] = func() _run.Run {
		return &DockerRun{
			Id:   "",
			Path: "",
		}
	}
}

// DockerRun implements task.Run
type DockerRun struct {
	Path string `json:"path"`
	Id   string `json:"id"`
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
	return err
}

func (d *DockerRun) Wait(ctx context.Context) (_status.Status, error) {
	cli, err := client.NewEnvClient() // FIXME use a singleton
	if err != nil {
		return _status.Error, err
	}
	waitC, errC := cli.ContainerWait(ctx, d.Id, "")
	loop := true
	var status _status.Status
	for loop {
		select {
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
	if status != 0 {
		// FIXME `docker-compose down`
		err = cli.ContainerKill(context.TODO(), d.Id, "KILL")
		if err != nil {
			return _status.Error, err
		}
		return status, nil
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
	return status, nil
}
