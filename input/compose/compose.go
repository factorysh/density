package compose

import (
	"errors"
	"fmt"
	"time"

	cmps "github.com/factorysh/density/compose"
	"github.com/factorysh/density/task"
)

func TaskFromCompose(com *cmps.Compose) (*task.Task, error) {
	cfgRaw, ok := com.X["x-batch"]
	if !ok {
		return nil, errors.New("Where is my x-batch?")
	}
	cfg, ok := cfgRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Wrong x-batch type: %v", cfg)
	}
	t := task.New()
	t.Action = com
	retry, ok := cfg["retry"]
	if ok {
		rr, ok := retry.(int)
		if !ok {
			return nil, fmt.Errorf("Bad retry type: %v", retry)
		}
		t.Retry = rr
	}
	maxExTime, ok := cfg["max_execution_time"].(string)
	if ok {
		mm, err := time.ParseDuration(maxExTime)
		if err != nil {
			return nil, err
		}
		t.MaxExectionTime = mm
	}

	every, ok := cfg["every"].(string)
	if ok {
		ee, err := time.ParseDuration(every)
		if err != nil {
			return nil, err
		}
		t.Every = ee
	}

	cron, ok := cfg["cron"].(string)
	if ok {
		_, err := task.Parser.Parse(cron)
		if err != nil {
			return nil, err
		}
		t.Cron = cron
	}

	return t, nil
}
