package compartment

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	CompartmentId uuid.UUID `json:"compartment_id"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	Type        byte   `json:"type"`
	Capacity    uint32 `json:"capacity"`
}

type DeletedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
}
