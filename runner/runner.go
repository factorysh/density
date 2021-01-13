package runner

import (
	"os"
	"path"

	"github.com/factorysh/batch-scheduler/task"
	"github.com/factorysh/batch-scheduler/task/run"
)

type Runner struct {
	Home string
}

func New(home string) *Runner {
	return &Runner{home}
}

// Up a Task
func (c *Runner) Up(task *task.Task) (run.Run, error) {
	pwd := path.Join(c.Home, task.Id.String())
	err := os.Mkdir(pwd, 0750)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	// FIXME add some late environments
	return task.Action.Up(pwd, task.Environments)
}
