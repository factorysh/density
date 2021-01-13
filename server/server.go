package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/factorysh/batch-scheduler/config"
	handlers "github.com/factorysh/batch-scheduler/handlers/api"
	"github.com/factorysh/batch-scheduler/runner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/store"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// Server struct containing config
type Server struct {
	Scheduler *scheduler.Scheduler
	AuthKey   string
	Addr      string
}

// Initialize server instance
func New() *Server {

	s := &Server{}
	var found bool

	if s.AuthKey, found = os.LookupEnv("AUTH_KEY"); !found {
		log.Fatal("Server can't start without an authentication key (`AUTH_KEY` env variable)")
	}

	err := config.EnsureDirs()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: dynamic ressource parameters (env, file, whatever)
	// FIXME where is my home?
	// TODO: storage kind and path from env
	// Plug bbolt with violence
	store, err := store.NewBoltStore("/tmp/batch.store")
	if err != nil {
		log.Fatal(err)
	}
	s.Scheduler = scheduler.New(scheduler.NewResources(2, 512*16), runner.New("/tmp"), store)

	var ok bool
	if s.Addr, ok = os.LookupEnv("LISTEN"); !ok {
		s.Addr = "localhost:8042"
	}

	return s
}

// Run starts this server instance
func (s *Server) Run(ctx context.Context) {

	ctxScheduler, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	err := s.Scheduler.Load()
	if err != nil {
		log.Fatal(err)
	}

	go s.Scheduler.Start(ctxScheduler)

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	server := &http.Server{
		Addr:    s.Addr,
		Handler: sentryHandler.HandleFunc(handlers.MuxAPI(s.Scheduler, s.AuthKey)),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancelShutdown()
		server.Shutdown(ctxShutdown)
		cancelShutdown()
	}
}
