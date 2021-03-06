package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	rawCompose "github.com/factorysh/density/compose"
	"github.com/factorysh/density/input/compose"
	"github.com/factorysh/density/owner"
	"github.com/factorysh/density/task"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// JOB is used as key in map of http vars
const JOB = "job"

// MAXFORMMEM is used to setup max form memory limit
const MAXFORMMEM = 1024

// HandleGetTasks handles a get on /schedules endpoint
func (a *API) HandleGetTasks(u *owner.Owner, w http.ResponseWriter,
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
			ts = a.schd.Filter(o)
		} else {
			// request all
			ts = a.schd.List()
		}
	} else {
		// used context information to get current user name
		ts = a.schd.Filter(u.Name)
	}

	return ts, nil
}

// HandlePostTasks handles a post on /tasks endpoint
func (a *API) HandlePostTasks(u *owner.Owner,
	w http.ResponseWriter, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	o, explicit := vars[owner.OWNER]

	t := new(task.Task)

	switch r.Header.Get("Content-Type") {
	case "application/json":
		err := json.NewDecoder(r.Body).Decode(t)
		r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, err
		}
	default:

		err := r.ParseMultipartForm(1 << 20)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, err
		}

		file, _, err := r.FormFile("docker-compose")
		defer file.Close()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, err
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, err
		}
		myCompose := rawCompose.NewCompose()
		err = yaml.Unmarshal(content, myCompose)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, err
		}

		err = myCompose.Validate()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, err
		}

		t, err = compose.TaskFromCompose(myCompose)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, err
		}
	}
	errs := a.validator.ValidateTask(t)
	if errs != nil && len(errs) > 0 {
		fmt.Println("Validate errors", errs)
		w.WriteHeader(400)
		errz := make([]string, len(errs))
		for i := 0; i < len(errs); i++ {
			errz[i] = errs[i].Error()
		}
		json.NewEncoder(w).Encode(errz)
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
	_, err := a.schd.Add(t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	fmt.Println(a.schd.List())

	w.WriteHeader(http.StatusCreated)
	return t, err
}

// HandleDeleteTasks handle a delete on schedules
func (a *API) HandleDeleteTasks(u *owner.Owner,
	w http.ResponseWriter, r *http.Request) (interface{}, error) {
	params := r.URL.Query()
	vars := mux.Vars(r)
	j, _ := vars[JOB]

	uuid, err := uuid.Parse(j)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	if _, wait := params["wait_for"]; wait {
		err := a.schd.Cancel(uuid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	}

	go a.schd.Cancel(uuid)
	w.WriteHeader(http.StatusAccepted)

	return nil, nil
}
