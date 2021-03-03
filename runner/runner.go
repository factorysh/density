package runner

import (
	"os"
	"path"

	"github.com/factorysh/density/task"
	_task "github.com/factorysh/density/task"
	"github.com/factorysh/density/task/run"
)

type Runner struct {
	home      string
	recompose *task.Recomposator
}

func New(home string, recompose *task.Recomposator) *Runner {
	return &Runner{
		home:      home,
		recompose: recompose,
	}
}

// Up a Task
func (c *Runner) Up(task *_task.Task) (run.Run, error) {
	pwd := path.Join(c.home, task.Id.String())
	err := os.Mkdir(pwd, 0750)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	var action _task.Action
	if c.recompose != nil {
		action, err = c.recompose.RecomposeAction(task.Action)
		if err != nil {
			return nil, err
		}
	} else {
		action = task.Action
	}
	// FIXME add some late environments
	return action.Up(pwd, task.Environments)
}

// GetHome fetch current data dir for runner
func (c *Runner) GetHome() string {
	return c.home
}
