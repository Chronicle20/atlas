package data

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopic        topic.Token = "EVENT_TOPIC_DATA"
	EventTypeDataUpdated             = "DATA_UPDATED"

	WorkerMonster = "MONSTER"
)

type event[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type dataUpdatedEventBody struct {
	TenantId    string `json:"tenantId"`
	Worker      string `json:"worker"`
	CompletedAt string `json:"completedAt"`
}
