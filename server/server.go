package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/factorysh/batch-scheduler/config"
	handlers "github.com/factorysh/batch-scheduler/handlers/api"
	"github.com/factorysh/batch-scheduler/middlewares"
	"github.com/factorysh/batch-scheduler/runner/compose"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/store"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
)

// Server struct containing config
type Server struct {
	API       *http.Server
	Done      chan (os.Signal)
	Router    *mux.Router
	Scheduler *scheduler.Scheduler
	AuthKey   string
}

// Initialize server instance
func (s *Server) Initialize() {

	var found bool

	if s.AuthKey, found = os.LookupEnv("AUTH_KEY"); !found {
		log.Fatal("Server can't start without an authentication key (`AUTH_KEY` env variable)")
	}

	err := config.EnsureDirs()
	if err != nil {
		log.Fatal(err)
	}

	s.Done = make(chan os.Signal, 1)
	signal.Notify(s.Done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// TODO: dynamic ressource parameters (env, file, whatever)
	// FIXME where is my home?
	s.Scheduler = scheduler.New(scheduler.NewResources(2, 512*16), compose.New("/tmp"), store.NewMemoryStore())

	// TODO: handle context
	go s.Scheduler.Start(context.Background())

	s.routes()

}

func (s *Server) routes() {

	// Create an instance of sentryhttp
	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	stack := func(f http.HandlerFunc) http.HandlerFunc {
		return middlewares.Auth(s.AuthKey, sentryHandler.HandleFunc(f))
	}

	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/api/schedules/{owner}", stack(handlers.HandleGetSchedules(s.Scheduler))).Methods(http.MethodGet)
	s.Router.HandleFunc("/api/schedules", stack(handlers.HandleGetSchedules(s.Scheduler))).Methods(http.MethodGet)
	s.Router.HandleFunc("/api/schedules", stack(handlers.HandlePostSchedules(s.Scheduler))).Methods(http.MethodPost)
	s.Router.HandleFunc("/api/schedules/{owner}", stack(handlers.HandlePostSchedules(s.Scheduler))).Methods(http.MethodPost)
	s.Router.HandleFunc("/api/schedules/{job}", stack(handlers.HandleDeleteSchedules(s.Scheduler))).Methods(http.MethodDelete)

}

// Run starts this server instance
func (s *Server) Run() {

	var port string
	var ok bool

	if port, ok = os.LookupEnv("PORT"); !ok {
		port = "8042"
	}

	s.API = &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: s.Router,
	}

	go func() {
		if err := s.API.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
}
