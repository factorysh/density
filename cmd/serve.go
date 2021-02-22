package cmd

import (
	"context"
	"fmt"
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
	Long:  ``,
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

		s := server.New()

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
