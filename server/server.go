package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/factorysh/density/compose"
	handlers "github.com/factorysh/density/handlers/api"
	"github.com/factorysh/density/runner"
	"github.com/factorysh/density/scheduler"
	"github.com/factorysh/density/store"
	"github.com/factorysh/density/task"
	"github.com/factorysh/density/version"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
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
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(`
		____                  _ _
		|  _ \  ___ _ __  ___(_) |_ _   _
		| | | |/ _ \ '_ \/ __| | __| | | |
		| |_| |  __/ | | \__ \ | |_| |_| |
		|____/ \___|_| |_|___/_|\__|\__, |
		                             |___/
		`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(version.Version()))
	}).Methods(http.MethodGet)
	v := &task.Validator{
		Validators: map[string]map[string]interface{}{
			"compose": compose.StandardConfig,
		},
	}
	err = v.Register()
	if err != nil { // FIXME it's ugly
		panic(err)
	}
	handlers.RegisterAPI(router.PathPrefix("/api").Subrouter(), s.Scheduler, v, s.AuthKey)
	server := &http.Server{
		Addr:    s.Addr,
		Handler: sentryHandler.HandleFunc(router.ServeHTTP),
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
