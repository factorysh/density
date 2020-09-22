package main

import (
	"context"
	"log"
	"time"

	"github.com/factorysh/batch-scheduler/action"
	"github.com/factorysh/batch-scheduler/server"
)

func main() {

	err := action.EnsureBin()
	if err != nil {
		log.Fatal(err)
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
