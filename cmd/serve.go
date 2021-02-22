package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"

	"github.com/factorysh/batch-scheduler/compose"
	"github.com/factorysh/batch-scheduler/server"
	"github.com/factorysh/batch-scheduler/version"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve REST API",
	Long: `
	Sentry is used if SENTRY_DSN env is set.
	LISTEN
	AUTH_KEY
	DATA_DIR
	CPU
	RAM
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		err := compose.EnsureBin()
		if err != nil {
			return err
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
				return err
			}
			// Flush buffered events before the program terminates.
			// Set the timeout to the maximum duration the program can afford to wait.
			defer sentry.Flush(2 * time.Second)
			sentry.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("service", "batch-scheduler")
			})
		}

		authKey := os.Getenv("AUTH_KEY")
		if authKey == "" {
			log.Fatal("Server can't start without an authentication key (`AUTH_KEY` env variable)")
		}

		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "/tmp/batch-scheduler"
		}

		addr := os.Getenv("LISTEN")
		if addr == "" {
			addr = "localhost:8042"
		}
		cpu := 2
		ram := 8 * 1024

		s, err := server.New(addr, dataDir, authKey, cpu, ram)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		fmt.Println("Listening", s.Addr)
		go s.Run(ctx)
		select {
		case <-done:
			fmt.Println("Bye")
			cancel()
		}
		return nil
	},
}
