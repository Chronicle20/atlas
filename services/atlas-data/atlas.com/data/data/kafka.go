package data

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_DATA"
)

const (
	CommandStartWorker = "START_WORKER"
)

const (
	EnvEventTopic topic.Token = "EVENT_TOPIC_DATA"
)

const (
	EventTypeDataUpdated = "DATA_UPDATED"
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
