package run

type Status int

const (
	Running = iota
	Paused
	Restarting
	Exited
	Dead
	Unkown
)
