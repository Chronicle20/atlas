package data

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_DATA"
)

const (
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
