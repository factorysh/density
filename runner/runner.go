package runner

import (
	"context"
	"os"
	"path"

	"github.com/factorysh/batch-scheduler/task"
)

type Runner struct {
	Home string
}

func New(home string) *Runner {
	return &Runner{home}
}

// Up a Task
func (c *Runner) Up(ctx context.Context, task *task.Task) error {
	pwd := path.Join(c.Home, task.Id.String())
	err := os.Mkdir(pwd, 0750)
	if err != nil && os.IsNotExist(err) {
		return err
	}
	env := map[string]string{
		"BASH_HELLO": "World",
	}
	return task.Action.Run(ctx, pwd, env)
}
