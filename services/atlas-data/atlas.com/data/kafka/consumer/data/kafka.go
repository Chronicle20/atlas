package data

const (
	EnvCommandTopic    = "COMMAND_TOPIC_DATA"
	CommandStartWorker = "START_WORKER"
)

type command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type startWorkerCommandBody struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
