package data

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopic    topic.Token = "COMMAND_TOPIC_DATA"
	CommandStartWorker             = "START_WORKER"

	EnvEventTopic        topic.Token = "EVENT_TOPIC_DATA"
	EventTypeDataUpdated             = "DATA_UPDATED"
)

type command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type startWorkerCommandBody struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type event[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type dataUpdatedEventBody struct {
	TenantId    string `json:"tenantId"`
	Worker      string `json:"worker"`
	CompletedAt string `json:"completedAt"`
}
