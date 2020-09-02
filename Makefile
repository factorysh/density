.PHONY: build test generate

bin:
	mkdir -p bin

run:
	./bin/batch-scheduler

build: bin
	go build -o bin/batch-scheduler cmd/batch-scheduler.go

test:
	go test -v -cover -timeout 30s github.com/factorysh/batch-scheduler/scheduler

generate:
	go generate ./task
