package ban

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_BAN"
)

const (
	CommandTypeCreate = "CREATE"
	CommandTypeDelete = "DELETE"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_BAN_STATUS"
)

const (
	EventStatusCreated = "CREATED"
	EventStatusDeleted = "DELETED"
	EventStatusExpired = "EXPIRED"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type CreateCommandBody struct {
	BanType    byte      `json:"banType"`
	Value      string    `json:"value"`
	Reason     string    `json:"reason"`
	ReasonCode byte      `json:"reasonCode"`
	Permanent  bool      `json:"permanent"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IssuedBy   string    `json:"issuedBy"`
}

type DeleteCommandBody struct {
	BanId uint32 `json:"banId"`
}

type StatusEvent struct {
	BanId  uint32 `json:"banId"`
	Status string `json:"status"`
}
