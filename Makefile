test:
	go test -v -cover -timeout 30s github.com/factorysh/batch-scheduler/scheduler

generate:
	go generate github.com/factorysh/batch-scheduler/scheduler