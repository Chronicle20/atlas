package compartment

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeCreated = "CREATED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Type     byte   `json:"type"`
	Capacity uint32 `json:"capacity"`
}
