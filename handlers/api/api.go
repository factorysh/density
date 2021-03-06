package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/factorysh/density/middlewares"
	"github.com/factorysh/density/owner"
	"github.com/factorysh/density/scheduler"
	"github.com/factorysh/density/task"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

type API struct {
	schd      *scheduler.Scheduler
	validator *task.Validator
	authKey   string
}

func RegisterAPI(router *mux.Router, schd *scheduler.Scheduler, validator *task.Validator, authKey string) {
	api := &API{
		schd:      schd,
		validator: validator,
		authKey:   authKey,
	}
	router.Use(middlewares.Auth(authKey))
	router.HandleFunc("/tasks/{owner}", api.wrapMyHandler(api.HandleGetTasks)).Methods(http.MethodGet)
	router.HandleFunc("/task/{uuid}", api.wrapMyHandler(api.HandleGetTask)).Methods(http.MethodGet)
	router.HandleFunc("/tasks", api.wrapMyHandler(api.HandleGetTasks)).Methods(http.MethodGet)
	router.HandleFunc("/tasks", api.wrapMyHandler(api.HandlePostTasks)).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{owner}", api.wrapMyHandler(api.HandlePostTasks)).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{job}", api.wrapMyHandler(api.HandleDeleteTasks)).Methods(http.MethodDelete)
}

func (a *API) wrapMyHandler(handler func(*owner.Owner, http.ResponseWriter,
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
		data, err := handler(u, w, r)
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
