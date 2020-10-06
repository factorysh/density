package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/factorysh/batch-scheduler/action"
	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// JOB is used as key in map of http vars
const JOB = "job"

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(schd *scheduler.Scheduler) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var ts []*task.Task
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
		var desc action.Description
		var t task.Task

		vars := mux.Vars(r)
		o, explicit := vars[owner.OWNER]

		u, err := owner.FromCtx(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		// get job description from body
		err = json.NewDecoder(r.Body).Decode(&desc)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		a, err := action.NewAction(desc)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		_, err = a.Validate()
		if err != nil {
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

		// add tasks to current tasks
		_, err = schd.Add(&t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		fmt.Println(schd.List())

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
func HandleDeleteSchedules(schd *scheduler.Scheduler) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		j, _ := vars[JOB]

		uuid, err := uuid.Parse(j)
		err = schd.Cancel(uuid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusNoContent)

	}

}
