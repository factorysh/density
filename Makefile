GIT_VERSION?=$(shell git describe --tags --always --abbrev=42 --dirty)
.PHONY: build test generate

default: build

build: bin
	go build \
		-ldflags "-X github.com/factorysh/density/version.version=$(GIT_VERSION)" \
		-o bin/density \
		main.go

bin:
	mkdir -p bin

AUTH_KEY:=debug

run:
	AUTH_KEY=${AUTH_KEY} ./bin/density serve

test:
	docker system prune -f
	go test -cover -timeout 120s \
	github.com/factorysh/density/store \
	github.com/factorysh/density/todo \
	github.com/factorysh/density/compose \
	github.com/factorysh/density/task \
	github.com/factorysh/density/task/compose \
	github.com/factorysh/density/task/status \
	github.com/factorysh/density/runner \
	github.com/factorysh/density/pubsub \
	github.com/factorysh/density/scheduler \
	github.com/factorysh/density/network \
	github.com/factorysh/density/middlewares

generate:
	go get -u golang.org/x/tools/cmd/stringer
	go generate ./task

clean:
	rm -rf bin
