package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus       = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin      = "LOGIN"
	EventCharacterStatusTypeLogout     = "LOGOUT"
	EventCharacterStatusTypeMapChanged = "MAP_CHANGED"

	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeMeso = "REQUEST_CHANGE_MESO"

	StatusEventTypeMesoChanged       = "MESO_CHANGED"
	StatusEventTypeError             = "ERROR"
	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventLogoutBody struct {
}

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestChangeMesoBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}

type MesoChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}

type StatusEventMesoErrorBody struct {
	Error  string `json:"error"`
	Amount int32  `json:"amount"`
}
