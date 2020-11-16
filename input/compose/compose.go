package compose

import (
	"errors"
	"fmt"

	rawCompose "github.com/factorysh/batch-scheduler/compose"
	"github.com/factorysh/batch-scheduler/task"
)

func TaskFromCompose(cmps *rawCompose.Compose) (*task.Task, error) {
	services, err := cmps.Services()
	if err != nil {
		return nil, err
	}
	cfgRaw, ok := services["x-batch"]
	if !ok {
		return nil, errors.New("Where is my x-batch?")
	}
	cfg, ok := cfgRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Wrong x-batch type: %v", cfg)
	}
	t := task.New()
	t.Action = cmps
	retry, ok := cfg["retry"]
	if ok {
		rr, ok := retry.(int)
		if !ok {
			return nil, fmt.Errorf("Bad retry type: %v", retry)
		}
		t.Retry = rr
	}
	return t, nil
}
