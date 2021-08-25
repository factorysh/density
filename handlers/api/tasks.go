package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/factorysh/density/claims"
	rawCompose "github.com/factorysh/density/compose"
	"github.com/factorysh/density/input/compose"
	"github.com/factorysh/density/task"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	zg "github.com/mattn/go-zglob"
	"gopkg.in/yaml.v3"
)

// JOB is used as key in map of http vars
const JOB = "job"

// MAXFORMMEM is used to setup max form memory limit
const MAXFORMMEM = 1024

// HandleGetTasks handles a get on /schedules endpoint
func (a *API) HandleGetTasks(c *claims.Claims, w http.ResponseWriter,
	r *http.Request) (interface{}, error) {
	labels := make(map[string]string)
	var ts []*task.Task
	// toSend array contains task with translated to task.Resp, removing private task fields
	toSend := []task.Resp{}
	vars := mux.Vars(r)
	o, filter := vars["owner"]

	// unpriviledged user can't request with a filter option
	if !c.Admin && filter {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, nil
	}

	for key, values := range r.URL.Query() {
		if len(values) > 1 {
			w.WriteHeader(http.StatusBadRequest)
			return nil, fmt.Errorf("http parameter %s is used multiple times", key)
		}

		labels[key] = values[0]
	}

	// if user is an admin
	if c.Admin {
		if filter || len(labels) > 1 {
			// request with a filter
			ts = a.schd.Filter(o, labels)
		} else {
			// request all
			ts = a.schd.List()
		}
	} else {
		// used context information to get current user name
		ts = a.schd.Filter(o, labels)
	}

	for _, t := range ts {
		toSend = append(toSend, t.ToTaskResp())
	}

	return toSend, nil
}

// HandlePostTasks handles a post on /tasks endpoint
func (a *API) HandlePostTasks(c *claims.Claims,
	w http.ResponseWriter, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	o, explicit := vars["owner"]

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

		var labels map[string]string
		// TODO: handle more values here ?
		rawLabels, ok := r.Form["labels"]
		if ok && len(rawLabels) >= 1 {
			err := json.Unmarshal([]byte(rawLabels[0]), &labels)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return nil, err
			}
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

		if len(labels) >= 1 {
			t.Labels = labels
		}
	}

	var errs []error
	for key, value := range t.Labels {
		if !task.IsLabelValid(key) {
			errs = append(errs, fmt.Errorf("Key `%v` do not respect labels policy", key))
		}
		if !task.IsLabelValid(value) {
			errs = append(errs, fmt.Errorf("Key `%v` do not respect labels policy", value))
		}
	}
	if errs != nil && len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		errz := make([]string, len(errs))
		for i := 0; i < len(errs); i++ {
			errz[i] = errs[i].Error()
		}
		json.NewEncoder(w).Encode(errz)
		return nil, fmt.Errorf("Labels errors %v", errs)
	}

	errs = a.validator.ValidateAction(t.Action)
	if errs != nil && len(errs) > 0 {
		fmt.Println("Validate errors", errs)
		w.WriteHeader(400)
		errz := make([]string, len(errs))
		for i := 0; i < len(errs); i++ {
			errz[i] = errs[i].Error()
		}
		json.NewEncoder(w).Encode(errz)
		return nil, fmt.Errorf("Validate errors %v", errs)
	}
	// unpriviledged user can't create explicit job
	if !c.Admin && explicit {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, nil
	}

	// if user is admin and request for an explicit task creation
	if c.Admin && explicit {
		// use parameter as owner
		t.Owner = o
	} else {
		// else, just use the user passed in the context
		t.Owner = c.Owner
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
func (a *API) HandleDeleteTasks(u *claims.Claims,
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

	go a.schd.Delete(uuid)
	w.WriteHeader(http.StatusAccepted)

	return nil, nil
}

// HandleGetVolumes handler a Get query to retrive file status from a task volume
func (a *API) HandleGetVolumes(c *claims.Claims, w http.ResponseWriter, r *http.Request) (interface{}, error) {

	vars := mux.Vars(r)
	jobID, ok := vars[JOB]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Missing job ID in request vars")

	}

	// fetch authorized path from token
	c, err := claims.FromCtx(r.Context())

	// if no token found return an error
	if c.Path == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Missing path claim in JWT token")
	}

	subPath, err := extractPathFromURL(r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	// join path parts to get full path
	fullPath := path.Join(a.GetDataDir(), jobID, "volumes", subPath)

	matching, err := zg.Match(string(c.Path), fullPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	if !matching {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, fmt.Errorf("Unauthorization triggered for user %s on path %s", c.Owner, fullPath)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return nil, err
	}

	w.Header().Del("content-type")

	http.ServeFile(w, r, fullPath)

	return nil, nil
}

func extractPathFromURL(p string) (string, error) {
	// /api/tasks/id/volume
	const urlShrink int = 5

	// consider everything after `volume/` the requested path
	// get the file and the path
	sub, file := path.Split(p)
	subpath := strings.Split(sub, "/")

	// ensure that path can be shrinked
	if len(subpath) < urlShrink {
		return "", fmt.Errorf("Path %s is not a valid path from url", p)
	}

	subpath = subpath[urlShrink:]
	subpath = append(subpath, file)

	return path.Join(subpath...), nil
}
