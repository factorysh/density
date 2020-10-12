.PHONY: build test generate

default: build

build: bin
	go build -o bin/batch-scheduler cmd/batch-scheduler.go

bin:
	mkdir -p bin

run:
	./bin/batch-scheduler

test:
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/scheduler
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/action

generate:
	go generate ./task

clean:
	rm -rf bin
