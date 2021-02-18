package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/factorysh/batch-scheduler/middlewares"
	"github.com/factorysh/batch-scheduler/owner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

func MuxAPI(schd *scheduler.Scheduler, authKey string) http.HandlerFunc {
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{owner}", wrapMyHandler(schd, HandleGetSchedules)).Methods(http.MethodGet)
	router.HandleFunc("/api/task/{uuid}", wrapMyHandler(schd, HandleGetSchedule)).Methods(http.MethodGet)
	router.HandleFunc("/api/tasks", wrapMyHandler(schd, HandleGetSchedules)).Methods(http.MethodGet)
	router.HandleFunc("/api/tasks", wrapMyHandler(schd, HandlePostSchedules)).Methods(http.MethodPost)
	router.HandleFunc("/api/tasks/{owner}", wrapMyHandler(schd, HandlePostSchedules)).Methods(http.MethodPost)
	router.HandleFunc("/api/tasks/{job}", wrapMyHandler(schd, HandleDeleteSchedules)).Methods(http.MethodDelete)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		middlewares.Auth(authKey, router.ServeHTTP).ServeHTTP(w, r)
	}
}

func wrapMyHandler(schd *scheduler.Scheduler, handler func(*scheduler.Scheduler, *owner.Owner, http.ResponseWriter,
	*http.Request) (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		hub := sentry.GetHubFromContext(r.Context())
		u, err := owner.FromCtx(r.Context())
		if err != nil {
			hub.CaptureException(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, err := handler(schd, u, w, r)
		if err != nil {
			// FIXME correct error handling
			if hub == nil {
				fmt.Println("Error:", err)
			} else {
				hub.CaptureException(err)
			}
			return
		}
		json.NewEncoder(w).Encode(data)
	}
}
