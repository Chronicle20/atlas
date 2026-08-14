package pet

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PET"
	CommandPetSpawn          = "SPAWN"
	CommandPetDespawn        = "DESPAWN"
	CommandPetAttemptCommand = "ATTEMPT_COMMAND"
	CommandAwardCloseness    = "AWARD_CLOSENESS"
	CommandAwardFullness     = "AWARD_FULLNESS"
	CommandAwardLevel        = "AWARD_LEVEL"
	CommandSetExclude        = "EXCLUDE"
	CommandPetEvolve         = "EVOLVE"
	CommandSetSkill          = "SET_SKILL"
	CommandPetRevive         = "REVIVE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ActorId       uint32    `json:"actorId"`
	PetId         uint32    `json:"petId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type SpawnCommandBody struct {
	Lead bool `json:"lead"`
}

type DespawnCommandBody struct{}

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

type EvolveCommandBody struct{}

// ReviveCommandBody restores a dried-up pet's lifespan. It carries NO
// expiration: atlas-pets derives it from the consumed item's own WZ data, so a
// forged command cannot dictate a lifespan. SourceTemplateId names the consumed
// Water of Life (classification 518). Command[E] already carries TransactionId,
// ActorId and PetId, so the body needs nothing else.
type ReviveCommandBody struct {
	SourceTemplateId uint32 `json:"sourceTemplateId"`
}

// SetSkillCommandBody carries a semantic pet skill key (atlas-constants
// pet/skill spelling) — never a client wire bit.
type SetSkillCommandBody struct {
	Skill   string `json:"skill"`
	Enabled bool   `json:"enabled"`
}

const (
	EnvCommandTopicMovement = "COMMAND_TOPIC_PET_MOVEMENT"
)

type MovementCommand struct {
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      _map.Id    `json:"mapId"`
	Instance   uuid.UUID  `json:"instance"`
	ObjectId   uint64     `json:"objectId"`
	ObserverId uint32     `json:"observerId"`
	X          int16      `json:"x"`
	Y          int16      `json:"y"`
	Stance     byte       `json:"stance"`
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
	StatusEventTypeEvolved          = "EVOLVED"
	StatusEventTypeFlagChanged      = "FLAG_CHANGED"
	StatusEventTypeRevived          = "REVIVED"
	StatusEventTypeReviveFailed     = "REVIVE_FAILED"

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

type CreatedStatusEventBody struct{}

type DeletedStatusEventBody struct{}

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
	// CashId is the pet's cash serial. atlas-channel needs it here because the
	// SPAWNED event is the only input to the PetActivated packet, and that packet
	// must carry the same serial the client sees in the pet's inventory slot --
	// CPet::GetItemSlot (GMS v83 @0x703af3) binds the two by that value alone.
	CashId uint64 `json:"cashId"`
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
	Slot          int8      `json:"slot"`
	Closeness     uint16    `json:"closeness"`
	Amount        int16     `json:"amount"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
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

type FlagChangedStatusEventBody struct {
	Slot int8   `json:"slot"`
	Flag uint16 `json:"flag"`
}

type EvolvedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	OldTemplateId uint32    `json:"oldTemplateId"`
	NewTemplateId uint32    `json:"newTemplateId"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// RevivedStatusEventBody reports a successful Water of Life revive. Expiration
// is the absolute new dry-up instant; Slot is unchanged by the revive (a doll
// stays unsummoned) and is carried only so consumers need no extra read.
type RevivedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Expiration    time.Time `json:"expiration"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// ReviveFailedStatusEventBody is a REAL terminal failure event, not a silent
// drop. By the time REVIVE runs the player's Water of Life is already
// destroyed by the saga's first step, so a timeout-length wait for the refund
// would read as a lost item; the saga accepts this event and compensates
// immediately.
type ReviveFailedStatusEventBody struct {
	Reason        string    `json:"reason"`
	TransactionId uuid.UUID `json:"transactionId"`
}
