package handlers

import (
	"net/http"

	"github.com/factorysh/batch-scheduler/middlewares"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/gorilla/mux"
)

func MuxAPI(schd *scheduler.Scheduler, authKey string) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/api/schedules/{owner}", HandleGetSchedules(schd)).Methods(http.MethodGet)
	router.HandleFunc("/api/schedules", HandleGetSchedules(schd)).Methods(http.MethodGet)
	router.HandleFunc("/api/schedules", HandlePostSchedules(schd)).Methods(http.MethodPost)
	router.HandleFunc("/api/schedules/{owner}", HandlePostSchedules(schd)).Methods(http.MethodPost)
	router.HandleFunc("/api/schedules/{job}", HandleDeleteSchedules(schd)).Methods(http.MethodDelete)
	return middlewares.Auth(authKey, router.ServeHTTP)
}
