package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/factorysh/batch-scheduler/compose"
	"github.com/factorysh/batch-scheduler/server"
	"github.com/factorysh/batch-scheduler/version"
)

func main() {

	err := compose.EnsureBin()
	if err != nil {
		log.Fatal("ensure bin:", err)
	}

	dsn := os.Getenv("SENTRY_DSN")
	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			// Either set your DSN here or set the SENTRY_DSN environment variable.
			Dsn: dsn,
			// Enable printing of SDK debug messages.
			// Useful when getting started or trying to figure something out.
			Debug:   true,
			Release: version.Version(),
		})
		if err != nil {
			log.Fatal(err)
		}
		// Flush buffered events before the program terminates.
		// Set the timeout to the maximum duration the program can afford to wait.
		defer sentry.Flush(2 * time.Second)
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("service", "batch-scheduler")
		})
	}

	var s server.Server

	s.Initialize()
	s.Run()

	<-s.Done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = s.API.Shutdown(ctx)
	defer func() {
		cancel()
	}()

	if err != nil {
		log.Fatal(err)
	}

}
