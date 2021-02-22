package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

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

// New initializes server instance
func New(addr, dataDir, authKey string, cpu, ram int) (*Server, error) {

	dataDir = strings.TrimRight(dataDir, "/")

	for _, sub := range [3]string{"validator", "wd", "store"} {
		err := os.MkdirAll(path.Join(dataDir, sub), 0755)
		if err != nil {
			return nil, err
		}
	}

	store, err := store.NewBoltStore(path.Join(dataDir, "store", "batch.store"))
	if err != nil {
		return nil, err
	}

	return &Server{
		AuthKey: authKey,
		Addr:    addr,
		Scheduler: scheduler.New(scheduler.NewResources(cpu, ram),
			runner.New(path.Join(dataDir, "wd")), store),
	}, nil
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
