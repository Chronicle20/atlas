package saga

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"time"
)

// Type represents the type of saga
type Type string

const (
	StorageTransaction Type = "storage_transaction"
)

// Saga represents the entire saga transaction
type Saga struct {
	TransactionId uuid.UUID   `json:"transactionId"` // Unique ID for the transaction
	SagaType      Type        `json:"sagaType"`      // Type of the saga
	InitiatedBy   string      `json:"initiatedBy"`   // Who initiated the saga (e.g., "STORAGE")
	Steps         []Step[any] `json:"steps"`         // List of steps in the saga
}

// Status represents the status of a saga step
type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Action represents an action type for saga steps
type Action string

const (
	AwardMesos          Action = "award_mesos"
	UpdateStorageMesos  Action = "update_storage_mesos"
)

// Step represents a single step within a saga
type Step[T any] struct {
	StepId    string    `json:"stepId"`    // Unique ID for the step
	Status    Status    `json:"status"`    // Status of the step
	Action    Action    `json:"action"`    // The Action to be taken
	Payload   T         `json:"payload"`   // Data required for the action
	CreatedAt time.Time `json:"createdAt"` // Timestamp of when the step was created
	UpdatedAt time.Time `json:"updatedAt"` // Timestamp of the last update to the step
}

// AwardMesosPayload is the payload for the award_mesos action
type AwardMesosPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`   // WorldId associated with the action
	ChannelId   byte     `json:"channelId"` // ChannelId associated with the action
	ActorId     uint32   `json:"actorId"`   // ActorId identifies who is giving/taking the mesos
	ActorType   string   `json:"actorType"` // ActorType identifies the type of actor (e.g., "STORAGE")
	Amount      int32    `json:"amount"`    // Amount of mesos to award (can be negative for deduction)
}

// UpdateStorageMesosPayload is the payload for the update_storage_mesos action
type UpdateStorageMesosPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId initiating the update
	AccountId   uint32 `json:"accountId"`   // AccountId that owns the storage
	WorldId     byte   `json:"worldId"`     // WorldId for the storage
	Operation   string `json:"operation"`   // Operation: "SET", "ADD", "SUBTRACT"
	Mesos       uint32 `json:"mesos"`       // Mesos amount
}
