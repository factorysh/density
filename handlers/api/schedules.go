package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	rawCompose "github.com/factorysh/batch-scheduler/compose"
	"github.com/factorysh/batch-scheduler/input/compose"
	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// JOB is used as key in map of http vars
const JOB = "job"

// MAXFORMMEM is used to setup max form memory limit
const MAXFORMMEM = 1024

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(schd *scheduler.Scheduler, u *owner.Owner, w http.ResponseWriter,
	r *http.Request) (interface{}, error) {
	var ts []*task.Task
	vars := mux.Vars(r)
	o, filter := vars[owner.OWNER]

	// unpriviledged user can't request with a filter option
	if !u.Admin && filter {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, nil
	}

	// if user is an admin
	if u.Admin {
		if filter {
			//  request with a filter
			ts = schd.Filter(o)
		} else {
			// request all
			ts = schd.List()
		}
	} else {
		// used context information to get current user name
		ts = schd.Filter(u.Name)
	}

	return ts, nil
}

// HandlePostSchedules handles a post on /schedules endpoint
func HandlePostSchedules(schd *scheduler.Scheduler, u *owner.Owner,
	w http.ResponseWriter, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	o, explicit := vars[owner.OWNER]

	err := r.ParseMultipartForm(1 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	file, _, err := r.FormFile("docker-compose")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	hub := sentry.GetHubFromContext(r.Context())
	if hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("docker-compose.yml", content)
		})
	}

	var myCompose rawCompose.Compose
	err = yaml.Unmarshal(content, &myCompose)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	err = myCompose.Validate()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	t, err := compose.TaskFromCompose(&myCompose)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	// unpriviledged user can't create explicit job
	if !u.Admin && explicit {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, nil
	}

	// if user is admin and request for an explicit task creation
	if u.Admin && explicit {
		// use parameter as owner
		t.Owner = o
	} else {
		// else, just use the user passed in the context
		t.Owner = u.Name
	}
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("task", t.Id)
		})
	}

	// add tasks to current tasks
	_, err = schd.Add(t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	fmt.Println(schd.List())

	w.WriteHeader(http.StatusCreated)
	return t, err
}

// HandleDeleteSchedules handle a delete on schedules
func HandleDeleteSchedules(schd *scheduler.Scheduler, u *owner.Owner,
	w http.ResponseWriter, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	j, _ := vars[JOB]

	uuid, err := uuid.Parse(j)
	err = schd.Cancel(uuid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}
