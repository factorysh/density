package compose

import (
	"context"
	"os"
	"path"

	"github.com/factorysh/batch-scheduler/task"
)

type ComposeRunner struct {
	Home string
}

func New(home string) *ComposeRunner {
	return &ComposeRunner{home}
}

// Up a Task
func (c *ComposeRunner) Up(ctx context.Context, task *task.Task) error {
	pwd := path.Join(c.Home, task.Id.String())
	err := os.Mkdir(pwd, 0750)
	if err != nil {
		return err
	}
	env := map[string]string{
		"BASH_HELLO": "World",
	}
	return task.Action.Run(ctx, pwd, env)
}
