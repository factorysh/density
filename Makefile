GIT_VERSION?=$(shell git describe --tags --always --abbrev=42 --dirty)
.PHONY: build test generate

default: build

build: bin
	go build \
		-ldflags "-X github.com:factorysh/batch-scheduler/version.version=$(GIT_VERSION)" \
		-o bin/batch-scheduler \
		cmd/batch-scheduler.go

bin:
	mkdir -p bin

run:
	./bin/batch-scheduler

test:
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/store
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/compose
	go test -cover -timeout 30s \
	github.com/factorysh/batch-scheduler/scheduler

generate:
	go generate ./task

clean:
	rm -rf bin
