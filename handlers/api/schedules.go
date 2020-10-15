package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/factorysh/batch-scheduler/action"
	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// JOB is used as key in map of http vars
const JOB = "job"

// MAXFORMMEM is used to setup max form memory limit
const MAXFORMMEM = 1024

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(schd *scheduler.Scheduler) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var ts []*task.Task
		vars := mux.Vars(r)
		o, filter := vars[owner.OWNER]
		hub := sentry.GetHubFromContext(r.Context())

		// get user from context
		u, err := owner.FromCtx(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		// unpriviledged user can't request with a filter option
		if !u.Admin && filter {
			w.WriteHeader(http.StatusUnauthorized)
			return
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

		json, err := json.Marshal(&ts)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(json)

	}

}

// HandlePostSchedules handles a post on /schedules endpoint
func HandlePostSchedules(schd *scheduler.Scheduler) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var t task.Task
		hub := sentry.GetHubFromContext(r.Context())

		vars := mux.Vars(r)
		o, explicit := vars[owner.OWNER]

		u, err := owner.FromCtx(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		err = r.ParseMultipartForm(1 << 20)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		file, _, err := r.FormFile("docker-compose")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("No docker-compose file found"))
			return
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("docker-compose.yml", content)
			})
		}

		a, err := action.NewAction(action.DockerCompose, content)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = a.Validate()
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		// unpriviledged user can't create explicit job
		if !u.Admin && explicit {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// if user is admin and request for an explicit task creation
		if u.Admin && explicit {
			// use parameter as owner
			t = task.NewTask(o, a)
		} else {
			// else, just use the user passed in the context
			t = task.NewTask(u.Name, a)
		}
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("task", t.Id)
			})
		}

		// add tasks to current tasks
		_, err = schd.Add(&t)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		fmt.Println(schd.List())

		json, err := json.Marshal(&t)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(json)

	}

}

// HandleDeleteSchedules handle a delete on schedules
func HandleDeleteSchedules(schd *scheduler.Scheduler) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		j, _ := vars[JOB]
		hub := sentry.GetHubFromContext(r.Context())

		uuid, err := uuid.Parse(j)
		err = schd.Cancel(uuid)
		if err != nil {
			if hub != nil {
				hub.CaptureException(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusNoContent)

	}

}
