package compose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/docker/docker/client"
	"github.com/factorysh/batch-scheduler/task"
)

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
	waitC, errC := cli.ContainerWait(ctx, d.Id, "")
	loop := true
	var status task.Status
	for loop {
		select {
		case <-waitC: // FIXME exitcode is get later
			loop = false
		case err := <-errC:
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					loop = false
					status = task.Timeout
				case context.Canceled:
					loop = false
					status = task.Canceled
				default:
					return task.Error, err
				}
			}
		}
	}
	if status != 0 {
		// FIXME `docker-compose down`
		err = cli.ContainerKill(context.TODO(), d.Id, "KILL")
		if err != nil {
			return task.Error, err
		}
		return status, nil
	}
	inspect, err := cli.ContainerInspect(context.TODO(), d.Id)
	if err != nil {
		return task.Error, err
	}
	status = task.Error
	if inspect.State.Status == "exited" {
		if inspect.State.ExitCode == 0 {
			status = task.Done
		}
	}
	return status, nil
}
