package ban

import "time"

const (
	EnvCommandTopic = "COMMAND_TOPIC_BAN"

	CommandTypeCreate = "CREATE"
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
