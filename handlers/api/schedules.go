package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/factorysh/batch-scheduler/task"
)

// HandleGetSchedules handles a get on /schedules endpoint
func HandleGetSchedules(tasks *task.Tasks) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ts := tasks.List()

		json, err := json.Marshal(&ts)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		w.WriteHeader(http.StatusOK)
		w.Write(json)

	}

}
