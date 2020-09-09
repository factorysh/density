package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/gorilla/mux"
)

// JOB is used as key in map of http vars
const JOB = "job"

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(tasks *task.Tasks) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var ts []task.Task
		vars := mux.Vars(r)
		o, filter := vars[owner.OWNER]

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
				ts = tasks.Filter(o)
			} else {
				// request all
				ts = tasks.List()
			}
		} else {
			// used context information to get current user name
			ts = tasks.Filter(u.Name)
		}

		json, err := json.Marshal(&ts)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(json)

	}

}

// HandlePostSchedules handles a post on /schedules endpoint
func HandlePostSchedules(tasks *task.Tasks) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var t task.Task
		vars := mux.Vars(r)
		o, explicit := vars[owner.OWNER]

		u, err := owner.FromCtx(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
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
			t = task.NewTask(o)
		} else {
			// else, just use the user passed in the context
			t = task.NewTask(u.Name)
		}

		// add tasks to current tasks
		tasks.Add(t)

		json, err := json.Marshal(&t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(json)

	}

}

// HandleDeleteSchedules handle a delete on schedules
func HandleDeleteSchedules(tasks *task.Tasks) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var index int
		var found bool

		vars := mux.Vars(r)
		j, _ := vars[JOB]

		for i, cur := range tasks.List() {
			if cur.Id.String() == j {
				found = true
				index = i
				break
			}
		}

		if !found {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := tasks.Kill(index)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusNoContent)

	}

}
