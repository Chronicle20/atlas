package ban

import "time"

const (
	EnvCommandTopic = "COMMAND_TOPIC_BAN"

	CommandTypeCreate = "CREATE"
	CommandTypeDelete = "DELETE"

	EnvEventTopicStatus = "EVENT_TOPIC_BAN_STATUS"
	EventStatusCreated  = "CREATED"
	EventStatusDeleted  = "DELETED"
	EventStatusExpired  = "EXPIRED"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type CreateCommandBody struct {
	BanType    byte   `json:"banType"`
	Value      string `json:"value"`
	Reason     string `json:"reason"`
	ReasonCode byte   `json:"reasonCode"`
	Permanent  bool   `json:"permanent"`
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
