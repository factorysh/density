package scheduler

type Process interface {
	Kill() error
}
