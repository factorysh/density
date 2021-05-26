package compose

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	_status "github.com/factorysh/density/task/status"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	cli, err := client.NewEnvClient()
	assert.NoError(t, err)
	containerResp, err := cli.ContainerCreate(context.TODO(), &container.Config{
		Image: "busybox",
		Cmd:   strslice.StrSlice{"sleep", "1"},
	}, &container.HostConfig{}, nil, "")
	assert.NoError(t, err)
	fmt.Println(containerResp.ID)
	dr := &DockerRun{
		Path:    "/tmp",
		Id:      containerResp.ID,
		Start:   time.Now(),
		Running: true,
	}
	err = cli.ContainerStart(context.TODO(), containerResp.ID, types.ContainerStartOptions{})
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	status, err := dr.Wait(ctx)
	assert.NoError(t, err)
	assert.True(t, time.Since(dr.Start) < 2*time.Second)
	assert.Equal(t, _status.Done, status)

	assert.Equal(t, 0, dr.ExitCode)
}
func TestRunAndTimeout(t *testing.T) {
	cli, err := client.NewEnvClient()
	assert.NoError(t, err)
	containerResp, err := cli.ContainerCreate(context.TODO(), &container.Config{
		Image: "busybox",
		Cmd:   strslice.StrSlice{"sleep", "30"},
	}, &container.HostConfig{}, nil, "")
	assert.NoError(t, err)
	fmt.Println(containerResp.ID)
	dr := &DockerRun{
		Path:    "/tmp",
		Id:      containerResp.ID,
		Start:   time.Now(),
		Running: true,
	}
	err = cli.ContainerStart(context.TODO(), containerResp.ID, types.ContainerStartOptions{})
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	status, err := dr.Wait(ctx)
	assert.NoError(t, err)
	assert.True(t, time.Since(dr.Start) < 2*time.Second)
	assert.NotEqual(t, _status.Done, status)
	assert.Equal(t, _status.Timeout, status)
	assert.NotEqual(t, 0, dr.ExitCode)
}
