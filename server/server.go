package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/factorysh/batch-scheduler/handlers/api"
	handlers "github.com/factorysh/batch-scheduler/handlers/api"
	"github.com/factorysh/batch-scheduler/middlewares"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/gorilla/mux"
)

// Server struct containing config
type Server struct {
	API     *http.Server
	Done    chan (os.Signal)
	Router  *mux.Router
	Tasks   *task.Tasks
	AuthKey string
}

// Initialize server instance
func (s *Server) Initialize() {

	var found bool

	if s.AuthKey, found = os.LookupEnv("AUTH_KEY"); !found {
		log.Fatal("Server can't start without an authentication key (`AUTH_KEY` env variable)")
	}

	s.Done = make(chan os.Signal, 1)
	signal.Notify(s.Done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	s.Tasks = task.NewTasks()

	s.routes()

}

func (s *Server) routes() {

	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/api/schedules/{owner}", middlewares.Auth(s.AuthKey, handlers.HandleGetSchedules(s.Tasks))).Methods(http.MethodGet)
	s.Router.HandleFunc("/api/schedules", middlewares.Auth(s.AuthKey, api.HandleGetSchedules(s.Tasks))).Methods(http.MethodGet)

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
