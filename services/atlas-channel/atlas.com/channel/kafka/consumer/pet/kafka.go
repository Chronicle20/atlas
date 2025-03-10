package pet

import "atlas-channel/movement"

const (
	EnvStatusEventTopic      = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeCreated   = "CREATED"
	StatusEventTypeDeleted   = "DELETED"
	StatusEventTypeSpawned   = "SPAWNED"
	StatusEventTypeDespawned = "DESPAWNED"
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
	Slot       byte   `json:"slot"`
	Level      byte   `json:"level"`
	Tameness   uint16 `json:"tameness"`
	Fullness   byte   `json:"fullness"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
}

const (
	EnvEventTopicMovement = "EVENT_TOPIC_PET_MOVEMENT"
)

type movementEvent struct {
	WorldId   byte              `json:"worldId"`
	ChannelId byte              `json:"channelId"`
	MapId     uint32            `json:"mapId"`
	PetId     uint64            `json:"petId"`
	Slot      byte              `json:"slot"`
	OwnerId   uint32            `json:"ownerId"`
	Movement  movement.Movement `json:"movement"`
}
