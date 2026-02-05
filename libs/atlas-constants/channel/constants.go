package channel

type Id byte

type StatusType string

const (
	StatusTypeStarted  StatusType = "STARTED"
	StatusTypeShutdown StatusType = "SHUTDOWN"
)
