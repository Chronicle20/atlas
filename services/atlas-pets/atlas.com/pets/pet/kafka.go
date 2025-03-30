package pet

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
	Closeness  uint16 `json:"closeness"`
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
	Closeness  uint16 `json:"closeness"`
	Fullness   byte   `json:"fullness"`
	OldSlot    int8   `json:"oldSlot"`
	Reason     string `json:"reason"`
}

type commandResponseStatusEventBody struct {
	Slot      int8   `json:"slot"`
	Closeness uint16 `json:"closeness"`
	Fullness  byte   `json:"fullness"`
	CommandId byte   `json:"commandId"`
	Success   bool   `json:"success"`
}

type closenessChangedStatusEventBody struct {
	Slot      int8   `json:"slot"`
	Closeness uint16 `json:"closeness"`
	Amount    int16  `json:"amount"`
}

type fullnessChangedStatusEventBody struct {
	Slot     int8 `json:"slot"`
	Fullness byte `json:"fullness"`
	Amount   int8 `json:"amount"`
}

type levelChangedStatusEventBody struct {
	Slot   int8 `json:"slot"`
	Level  byte `json:"level"`
	Amount int8 `json:"amount"`
}

type slotChangedStatusEventBody struct {
	OldSlot int8 `json:"oldSlot"`
	NewSlot int8 `json:"newSlot"`
}

type excludeChangedStatusEventBody struct {
	Items []uint32 `json:"items"`
}

const (
	EnvEventTopicMovement     = "EVENT_TOPIC_PET_MOVEMENT"
	MovementTypeNormal        = "NORMAL"
	MovementTypeTeleport      = "TELEPORT"
	MovementTypeStartFallDown = "START_FALL_DOWN"
	MovementTypeFlyingBlock   = "FLYING_BLOCK"
	MovementTypeJump          = "JUMP"
	MovementTypeStatChange    = "STAT_CHANGE"
)

type movementEvent struct {
	WorldId   byte     `json:"worldId"`
	ChannelId byte     `json:"channelId"`
	MapId     uint32   `json:"mapId"`
	PetId     uint64   `json:"petId"`
	Slot      int8     `json:"slot"`
	OwnerId   uint32   `json:"ownerId"`
	Movement  Movement `json:"movement"`
}

type Movement struct {
	StartX   int16     `json:"startX"`
	StartY   int16     `json:"startY"`
	Elements []Element `json:"elements"`
}

type Element struct {
	TypeStr     string `json:"typeStr"`
	TypeVal     byte   `json:"typeVal"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	MoveAction  byte   `json:"moveAction"`
	Stat        byte   `json:"stat"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	VX          int16  `json:"vX"`
	VY          int16  `json:"vY"`
	FH          int16  `json:"fh"`
	FHFallStart int16  `json:"fhFallStart"`
	XOffset     int16  `json:"xOffset"`
	YOffset     int16  `json:"yOffset"`
	TimeElapsed int16  `json:"timeElapsed"`
}
