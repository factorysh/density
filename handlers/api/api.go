package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/factorysh/density/middlewares"
	"github.com/factorysh/density/owner"
	"github.com/factorysh/density/scheduler"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

func RegisterAPI(router *mux.Router, schd *scheduler.Scheduler, authKey string) {
	router.Use(middlewares.Auth(authKey))
	router.HandleFunc("/tasks/{owner}", wrapMyHandler(schd, HandleGetSchedules)).Methods(http.MethodGet)
	router.HandleFunc("/task/{uuid}", wrapMyHandler(schd, HandleGetSchedule)).Methods(http.MethodGet)
	router.HandleFunc("/tasks", wrapMyHandler(schd, HandleGetSchedules)).Methods(http.MethodGet)
	router.HandleFunc("/tasks", wrapMyHandler(schd, HandlePostSchedules)).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{owner}", wrapMyHandler(schd, HandlePostSchedules)).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{job}", wrapMyHandler(schd, HandleDeleteSchedules)).Methods(http.MethodDelete)
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
