.PHONY: build test generate

bin:
	mkdir -p bin

run:
	./bin/batch-scheduler

build: bin
	go build -o bin/batch-scheduler cmd/batch-scheduler.go

test:
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/scheduler
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/action

generate:
	go generate ./task
