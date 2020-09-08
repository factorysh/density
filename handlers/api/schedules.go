package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/gorilla/mux"
)

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(tasks *task.Tasks) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var ts []task.Task
		vars := mux.Vars(r)
		o, filter := vars["owner"]

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

		// if user if an admin
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

		u, err := owner.FromCtx(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		t := task.Task{
			Owner: u.Name,
		}
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
