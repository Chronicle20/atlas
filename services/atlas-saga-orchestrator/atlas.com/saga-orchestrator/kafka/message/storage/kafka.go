package storage

import (
	"github.com/google/uuid"
	"time"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_STORAGE"
	EnvShowStorageCommandTopic = "COMMAND_TOPIC_STORAGE_SHOW"
	CommandTypeDeposit         = "DEPOSIT"
	CommandTypeWithdraw        = "WITHDRAW"
	CommandTypeUpdateMesos     = "UPDATE_MESOS"
	CommandTypeDepositRollback = "DEPOSIT_ROLLBACK"
	CommandTypeShowStorage     = "SHOW_STORAGE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// DepositBody contains the data needed to deposit an item into storage
type DepositBody struct {
	Slot          int16         `json:"slot"`
	TemplateId    uint32        `json:"templateId"`
	Expiration    time.Time     `json:"expiration"`
	ReferenceId   uint32        `json:"referenceId"`
	ReferenceType string        `json:"referenceType"`
	ReferenceData ReferenceData `json:"referenceData,omitempty"`
}

// ReferenceData contains the reference data for stackable items
type ReferenceData struct {
	Quantity uint32 `json:"quantity,omitempty"`
	OwnerId  uint32 `json:"ownerId,omitempty"`
	Flag     uint16 `json:"flag,omitempty"`
}

// WithdrawBody contains the data needed to withdraw an item from storage
type WithdrawBody struct {
	AssetId  uint32 `json:"assetId"`
	Quantity uint32 `json:"quantity,omitempty"`
}

// UpdateMesosBody contains the data needed to update mesos in storage
type UpdateMesosBody struct {
	Mesos     uint32 `json:"mesos"`
	Operation string `json:"operation"` // "SET", "ADD", "SUBTRACT"
}

// DepositRollbackBody contains the data needed to rollback a deposit
type DepositRollbackBody struct {
	AssetId uint32 `json:"assetId"`
}

const (
	EnvStatusEventTopic        = "EVENT_TOPIC_STORAGE_STATUS"
	StatusEventTypeDeposited   = "DEPOSITED"
	StatusEventTypeWithdrawn   = "WITHDRAWN"
	StatusEventTypeMesosUpdate = "MESOS_UPDATED"
	StatusEventTypeError       = "ERROR"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// DepositedEventBody contains the data for a deposited event
type DepositedEventBody struct {
	AssetId       uint32    `json:"assetId"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"templateId"`
	ReferenceId   uint32    `json:"referenceId"`
	ReferenceType string    `json:"referenceType"`
	Expiration    time.Time `json:"expiration"`
}

// WithdrawnEventBody contains the data for a withdrawn event
type WithdrawnEventBody struct {
	AssetId    uint32 `json:"assetId"`
	Slot       int16  `json:"slot"`
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity,omitempty"`
}

// MesosUpdatedEventBody contains the data for a mesos updated event
type MesosUpdatedEventBody struct {
	OldMesos uint32 `json:"oldMesos"`
	NewMesos uint32 `json:"newMesos"`
}

// ErrorEventBody contains the data for an error event
type ErrorEventBody struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// ShowStorageCommand is sent to the channel service to display storage UI
type ShowStorageCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	ChannelId     byte      `json:"channelId"`
	CharacterId   uint32    `json:"characterId"`
	NpcId         uint32    `json:"npcId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
}
