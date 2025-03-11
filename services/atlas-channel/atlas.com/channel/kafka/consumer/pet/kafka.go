package pet

import "atlas-channel/movement"

const (
	EnvStatusEventTopic            = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeSpawned         = "SPAWNED"
	StatusEventTypeDespawned       = "DESPAWNED"
	StatusEventTypeCommandResponse = "COMMAND_RESPONSE"
)

type statusEvent[E any] struct {
	PetId   uint64 `json:"petId"`
	OwnerId uint32 `json:"ownerId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createdStatusEventBody struct {
}

type deletedStatusEventBody struct {
}

type spawnedStatusEventBody struct {
	TemplateId uint32 `json:"templateId"`
	Name       string `json:"name"`
	Slot       int8   `json:"slot"`
	Level      byte   `json:"level"`
	Tameness   uint16 `json:"tameness"`
	Fullness   byte   `json:"fullness"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
	FH         int16  `json:"fh"`
}

type despawnedStatusEventBody struct {
	TemplateId uint32 `json:"templateId"`
	Name       string `json:"name"`
	Slot       int8   `json:"slot"`
	Level      byte   `json:"level"`
	Tameness   uint16 `json:"tameness"`
	Fullness   byte   `json:"fullness"`
}

type commandResponseStatusEventBody struct {
	Slot      int8   `json:"slot"`
	Tameness  uint16 `json:"tameness"`
	Fullness  byte   `json:"fullness"`
	CommandId byte   `json:"commandId"`
	Success   bool   `json:"success"`
}

const (
	EnvEventTopicMovement = "EVENT_TOPIC_PET_MOVEMENT"
)

type movementEvent struct {
	WorldId   byte              `json:"worldId"`
	ChannelId byte              `json:"channelId"`
	MapId     uint32            `json:"mapId"`
	PetId     uint64            `json:"petId"`
	Slot      int8              `json:"slot"`
	OwnerId   uint32            `json:"ownerId"`
	Movement  movement.Movement `json:"movement"`
}
