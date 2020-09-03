package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/factorysh/batch-scheduler/handlers/api"
	"github.com/factorysh/batch-scheduler/task"
	"github.com/gorilla/mux"
)

// Server struct containing config
type Server struct {
	API    *http.Server
	Done   chan (os.Signal)
	Router *mux.Router
	Tasks  *task.Tasks
}

// Initialize server instance
func (s *Server) Initialize() {

	s.Done = make(chan os.Signal, 1)
	signal.Notify(s.Done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	s.Tasks = task.NewTasks()

	s.routes()

}

func (s *Server) routes() {

	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/api/schedules/{owner}", api.HandleGetSchedules(s.Tasks)).Methods(http.MethodGet)
	s.Router.HandleFunc("/api/schedules", api.HandleGetSchedules(s.Tasks)).Methods(http.MethodGet)

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
