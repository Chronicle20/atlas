package pet

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_PET"
)

const (
	CommandPetSpawn          = "SPAWN"
	CommandPetDespawn        = "DESPAWN"
	CommandPetAttemptCommand = "ATTEMPT_COMMAND"
	CommandPetSetExclude     = "EXCLUDE"
	CommandPetRename         = "RENAME"
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

type DespawnCommandBody struct{}

type AttemptCommandCommandBody struct {
	CommandId byte `json:"commandId"`
	ByName    bool `json:"byName"`
}

type SetExcludeCommandBody struct {
	Items []uint32 `json:"items"`
}

// RenameCommandBody carries the new pet name. It is ALREADY normalized by the
// caller, but atlas-pets re-validates it regardless (PRD FR-5.6) — the channel
// is not trusted to have validated, and a crafted producer could publish
// straight to this topic.
type RenameCommandBody struct {
	Name string `json:"name"`
}

const (
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_PET_STATUS"
)

const (
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
	StatusEventTypeFlagChanged      = "FLAG_CHANGED"
	StatusEventTypeReviveFailed     = "REVIVE_FAILED"
	StatusEventTypeNameChanged      = "NAME_CHANGED"
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

type FlagChangedStatusEventBody struct {
	Slot int8   `json:"slot"`
	Flag uint16 `json:"flag"`
}

// ReviveFailedStatusEventBody reports that atlas-pets rejected a Water of Life
// revive after the item was already consumed. The saga refunds the item; this
// channel consumer is what tells the player why nothing happened.
type ReviveFailedStatusEventBody struct {
	Reason        string    `json:"reason"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// NameChangedStatusEventBody drives two consumers. atlas-channel needs Slot to
// address the clientbound PET_NAMECHANGE packet (the packet carries no pet id —
// it is routed by ownerId+slot). The orchestrator needs TransactionId to
// complete the rename_pet step. PreviousName is what the compensator re-applies
// if the consume step later fails.
type NameChangedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Name          string    `json:"name"`
	PreviousName  string    `json:"previousName"`
	TransactionId uuid.UUID `json:"transactionId"`
}
