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
		owner, filter := vars["owner"]

		if filter {
			ts = tasks.Filter(owner)
		} else {
			ts = tasks.List()
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
