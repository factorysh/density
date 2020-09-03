package handlers

import (
	"encoding/json"
	"net/http"

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
		}

		w.WriteHeader(http.StatusOK)
		w.Write(json)

	}

}
