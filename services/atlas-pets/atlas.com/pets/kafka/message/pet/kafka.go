package pet

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PET"
	CommandPetSpawn          = "SPAWN"
	CommandPetDespawn        = "DESPAWN"
	CommandPetAttemptCommand = "ATTEMPT_COMMAND"
	CommandAwardCloseness    = "AWARD_CLOSENESS"
	CommandAwardFullness     = "AWARD_FULLNESS"
	CommandAwardLevel        = "AWARD_LEVEL"
	CommandSetExclude        = "EXCLUDE"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint32 `json:"petId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type SpawnCommandBody struct {
	Lead bool `json:"lead"`
}

type DespawnCommandBody struct {
}

type AttemptCommandCommandBody struct {
	CommandId byte `json:"commandId"`
	ByName    bool `json:"byName"`
}

type AwardClosenessCommandBody struct {
	Amount uint16 `json:"amount"`
}

type AwardFullnessCommandBody struct {
	Amount byte `json:"amount"`
}

type AwardLevelCommandBody struct {
	Amount byte `json:"amount"`
}

type SetExcludeCommandBody struct {
	Items []uint32 `json:"items"`
}

const (
	EnvCommandTopicMovement = "COMMAND_TOPIC_PET_MOVEMENT"
)

type MovementCommand struct {
	WorldId    byte   `json:"worldId"`
	ChannelId  byte   `json:"channelId"`
	MapId      uint32 `json:"mapId"`
	ObjectId   uint64 `json:"objectId"`
	ObserverId uint32 `json:"observerId"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
}

const (
	EnvStatusEventTopic             = "EVENT_TOPIC_PET_STATUS"
	StatusEventTypeCreated          = "CREATED"
	StatusEventTypeDeleted          = "DELETED"
	StatusEventTypeSpawned          = "SPAWNED"
	StatusEventTypeDespawned        = "DESPAWNED"
	StatusEventTypeCommandResponse  = "COMMAND_RESPONSE"
	StatusEventTypeClosenessChanged = "CLOSENESS_CHANGED"
	StatusEventTypeFullnessChanged  = "FULLNESS_CHANGED"
	StatusEventTypeLevelChanged     = "LEVEL_CHANGED"
	StatusEventTypeSlotChanged      = "SLOT_CHANGED"
	StatusEventTypeExcludeChanged   = "EXCLUDE_CHANGED"

	DespawnReasonNormal  = "NORMAL"
	DespawnReasonHunger  = "HUNGER"
	DespawnReasonExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	PetId   uint32 `json:"petId"`
	OwnerId uint32 `json:"ownerId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type CreatedStatusEventBody struct {
}

type DeletedStatusEventBody struct {
}

type SpawnedStatusEventBody struct {
	TemplateId uint32 `json:"templateId"`
	Name       string `json:"name"`
	Slot       int8   `json:"slot"`
	Level      byte   `json:"level"`
	Closeness  uint16 `json:"closeness"`
	Fullness   byte   `json:"fullness"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
	FH         int16  `json:"fh"`
}

type DespawnedStatusEventBody struct {
	TemplateId uint32 `json:"templateId"`
	Name       string `json:"name"`
	Slot       int8   `json:"slot"`
	Level      byte   `json:"level"`
	Closeness  uint16 `json:"closeness"`
	Fullness   byte   `json:"fullness"`
	OldSlot    int8   `json:"oldSlot"`
	Reason     string `json:"reason"`
}

type CommandResponseStatusEventBody struct {
	Slot      int8   `json:"slot"`
	Closeness uint16 `json:"closeness"`
	Fullness  byte   `json:"fullness"`
	CommandId byte   `json:"commandId"`
	Success   bool   `json:"success"`
}

type ClosenessChangedStatusEventBody struct {
	Slot      int8   `json:"slot"`
	Closeness uint16 `json:"closeness"`
	Amount    int16  `json:"amount"`
}

type FullnessChangedStatusEventBody struct {
	Slot     int8 `json:"slot"`
	Fullness byte `json:"fullness"`
	Amount   int8 `json:"amount"`
}

type LevelChangedStatusEventBody struct {
	Slot   int8 `json:"slot"`
	Level  byte `json:"level"`
	Amount int8 `json:"amount"`
}

type SlotChangedStatusEventBody struct {
	OldSlot int8 `json:"oldSlot"`
	NewSlot int8 `json:"newSlot"`
}

type ExcludeChangedStatusEventBody struct {
	Items []uint32 `json:"items"`
}
