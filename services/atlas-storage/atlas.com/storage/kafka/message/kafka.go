package message

import (
	"github.com/google/uuid"
	"time"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_STORAGE"
	EnvEventTopic              = "EVENT_TOPIC_STORAGE_STATUS"
	EnvCommandTopicAssetExpire = "COMMAND_TOPIC_ASSET_EXPIRE"
)

// Command types
const (
	CommandTypeDeposit         = "DEPOSIT"
	CommandTypeWithdraw        = "WITHDRAW"
	CommandTypeUpdateMesos     = "UPDATE_MESOS"
	CommandTypeDepositRollback = "DEPOSIT_ROLLBACK"
	CommandTypeArrange         = "ARRANGE"
	CommandTypeShowStorage     = "SHOW_STORAGE"
	CommandTypeCloseStorage    = "CLOSE_STORAGE"
)

// Additional command topic for show/close storage operations
const (
	EnvShowStorageCommandTopic = "COMMAND_TOPIC_STORAGE_SHOW"
)

// Event types
const (
	StatusEventTypeDeposited           = "DEPOSITED"
	StatusEventTypeWithdrawn           = "WITHDRAWN"
	StatusEventTypeMesosUpdated        = "MESOS_UPDATED"
	StatusEventTypeArranged            = "ARRANGED"
	StatusEventTypeError               = "ERROR"
	StatusEventTypeProjectionCreated   = "PROJECTION_CREATED"
	StatusEventTypeProjectionDestroyed = "PROJECTION_DESTROYED"
	StatusEventTypeExpired             = "EXPIRED"
)

// Error codes
const (
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeOneOfAKind     = "ONE_OF_A_KIND"
	ErrorCodeGeneric        = "GENERIC"
)

// Command is the base command structure for storage operations
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEvent is the base event structure for storage status updates
type StatusEvent[E any] struct {
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
	AssetId         uint32 `json:"assetId"`
	TargetSlot      int16  `json:"targetSlot,omitempty"`
	Quantity        uint32 `json:"quantity,omitempty"`
	TargetStorageId string `json:"targetStorageId,omitempty"`
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

// ArrangeBody contains the data needed to arrange storage (empty for now)
type ArrangeBody struct {
}

// ArrangedEventBody contains the data for an arranged event
type ArrangedEventBody struct {
}

// ErrorEventBody contains the data for an error event
type ErrorEventBody struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message,omitempty"`
}

// ShowStorageCommand is received from the saga-orchestrator to track NPC context for storage
type ShowStorageCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	ChannelId     byte      `json:"channelId"`
	CharacterId   uint32    `json:"characterId"`
	NpcId         uint32    `json:"npcId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
}

// CloseStorageCommand is received when a character closes storage
type CloseStorageCommand struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
}

// ProjectionCreatedEventBody contains the data for a projection created event
type ProjectionCreatedEventBody struct {
	CharacterId uint32 `json:"characterId"`
	AccountId   uint32 `json:"accountId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	NpcId       uint32 `json:"npcId"`
}

// ProjectionDestroyedEventBody contains the data for a projection destroyed event
type ProjectionDestroyedEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

// ExpireCommand is received from atlas-asset-expiration to expire an item
type ExpireCommand struct {
	TransactionId  uuid.UUID `json:"transactionId"`
	CharacterId    uint32    `json:"characterId"`
	AccountId      uint32    `json:"accountId"`
	WorldId        byte      `json:"worldId"`
	AssetId        uint32    `json:"assetId"`
	TemplateId     uint32    `json:"templateId"`
	InventoryType  int8      `json:"inventoryType"`
	Slot           int16     `json:"slot"`
	ReplaceItemId  uint32    `json:"replaceItemId"`
	ReplaceMessage string    `json:"replaceMessage"`
	Source         string    `json:"source"` // "INVENTORY", "STORAGE", or "CASHSHOP"
}

// ExpiredStatusEventBody contains information about an expired item for client notification
type ExpiredStatusEventBody struct {
	IsCash         bool   `json:"isCash"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}
