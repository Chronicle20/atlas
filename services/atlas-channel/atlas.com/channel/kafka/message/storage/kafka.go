package storage

import "github.com/google/uuid"

const (
	EnvShowStorageCommandTopic = "COMMAND_TOPIC_STORAGE_SHOW"
	CommandTypeShowStorage     = "SHOW_STORAGE"
	CommandTypeCloseStorage    = "CLOSE_STORAGE"

	// Storage command topic for operations
	EnvCommandTopic        = "COMMAND_TOPIC_STORAGE"
	CommandTypeArrange     = "ARRANGE"
	CommandTypeUpdateMesos = "UPDATE_MESOS"

	// Mesos operations
	MesosOperationSet      = "SET"
	MesosOperationAdd      = "ADD"
	MesosOperationSubtract = "SUBTRACT"
)

// ShowStorageCommand is received from the saga-orchestrator to display storage UI
type ShowStorageCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	ChannelId     byte      `json:"channelId"`
	CharacterId   uint32    `json:"characterId"`
	NpcId         uint32    `json:"npcId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
}

// CloseStorageCommand is sent when a character closes storage
type CloseStorageCommand struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
}

// Command represents a storage command sent to the storage service
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ArrangeCommandBody contains data for the ARRANGE command
type ArrangeCommandBody struct {
}

// UpdateMesosCommandBody contains data for the UPDATE_MESOS command
type UpdateMesosCommandBody struct {
	Mesos     uint32 `json:"mesos"`
	Operation string `json:"operation"` // "SET", "ADD", "SUBTRACT"
}

const (
	// Storage status event topic
	EnvEventTopicStatus                   = "EVENT_TOPIC_STORAGE_STATUS"
	EnvEventTopicStorageCompartmentStatus = "EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS"
	StatusEventTypeDeposited              = "DEPOSITED"
	StatusEventTypeWithdrawn              = "WITHDRAWN"
	StatusEventTypeMesosUpdated           = "MESOS_UPDATED"
	StatusEventTypeArranged               = "ARRANGED"
	StatusEventTypeError                  = "ERROR"
	StatusEventTypeProjectionCreated      = "PROJECTION_CREATED"
	StatusEventTypeProjectionDestroyed    = "PROJECTION_DESTROYED"

	// Storage compartment event types
	StatusEventTypeCompartmentAccepted = "ACCEPTED"
	StatusEventTypeCompartmentReleased = "RELEASED"

	// Error codes
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeOneOfAKind     = "ONE_OF_A_KIND"
	ErrorCodeGeneric        = "GENERIC"
)

// StatusEvent represents a storage status event
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// MesosUpdatedEventBody contains the data for a mesos updated event
type MesosUpdatedEventBody struct {
	OldMesos uint32 `json:"oldMesos"`
	NewMesos uint32 `json:"newMesos"`
}

// ArrangedEventBody contains the data for an arranged event
type ArrangedEventBody struct {
}

// ErrorEventBody contains the data for an error event
type ErrorEventBody struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message,omitempty"`
}

// StorageCompartmentEvent represents a storage compartment status event
type StorageCompartmentEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	AccountId   uint32 `json:"accountId"`
	CharacterId uint32 `json:"characterId,omitempty"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// CompartmentAcceptedEventBody contains the data for an ACCEPTED event
type CompartmentAcceptedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	Slot          int16     `json:"slot"`
	InventoryType byte      `json:"inventoryType"`
}

// CompartmentReleasedEventBody contains the data for a RELEASED event
type CompartmentReleasedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	InventoryType byte      `json:"inventoryType"`
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
