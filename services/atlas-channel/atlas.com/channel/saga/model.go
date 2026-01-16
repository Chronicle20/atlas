package saga

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"time"
)

// Type represents the type of saga
type Type string

const (
	InventoryTransaction Type = "inventory_transaction"
	StorageOperation     Type = "storage_operation"
	CashShopOperation    Type = "cash_shop_operation"
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
	AwardMesos           Action = "award_mesos"
	UpdateStorageMesos   Action = "update_storage_mesos"
	AwardAsset           Action = "award_asset"
	DestroyAsset         Action = "destroy_asset"
	DepositToStorage     Action = "deposit_to_storage"
	TransferToStorage    Action = "transfer_to_storage"    // High-level action for inventory -> storage
	WithdrawFromStorage  Action = "withdraw_from_storage"  // High-level action for storage -> inventory
	TransferToCashShop   Action = "transfer_to_cash_shop"  // High-level action for inventory -> cash shop
	WithdrawFromCashShop Action = "withdraw_from_cash_shop" // High-level action for cash shop -> inventory
	AcceptToStorage      Action = "accept_to_storage"      // Internal (created by saga-orchestrator)
	ReleaseFromCharacter Action = "release_from_character" // Internal (created by saga-orchestrator)
	AcceptToCharacter    Action = "accept_to_character"    // Internal (created by saga-orchestrator)
	ReleaseFromStorage   Action = "release_from_storage"   // Internal (created by saga-orchestrator)
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
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ActorId     uint32     `json:"actorId"`     // ActorId identifies who is giving/taking the mesos
	ActorType   string     `json:"actorType"`   // ActorType identifies the type of actor (e.g., "STORAGE")
	Amount      int32      `json:"amount"`      // Amount of mesos to award (can be negative for deduction)
}

// UpdateStorageMesosPayload is the payload for the update_storage_mesos action
type UpdateStorageMesosPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId initiating the update
	AccountId   uint32 `json:"accountId"`   // AccountId that owns the storage
	WorldId     byte   `json:"worldId"`     // WorldId for the storage
	Operation   string `json:"operation"`   // Operation: "SET", "ADD", "SUBTRACT"
	Mesos       uint32 `json:"mesos"`       // Mesos amount
}

// AwardAssetPayload is the payload for the award_asset action
type AwardAssetPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	Item        ItemPayload `json:"item"`        // Item to award
}

// ItemPayload represents an individual item in a transaction
type ItemPayload struct {
	TemplateId uint32    `json:"templateId"`           // TemplateId of the item
	Quantity   uint32    `json:"quantity"`             // Quantity of the item
	Expiration time.Time `json:"expiration,omitempty"` // Expiration time for the item (zero value = no expiration)
}

// DestroyAssetPayload is the payload for the destroy_asset action
type DestroyAssetPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	TemplateId  uint32 `json:"templateId"`  // TemplateId of the item to destroy
	Quantity    uint32 `json:"quantity"`    // Quantity of the item to destroy (ignored if RemoveAll is true)
	RemoveAll   bool   `json:"removeAll"`   // If true, remove all instances of the item regardless of Quantity
}

// DepositToStoragePayload is the payload for the deposit_to_storage action
type DepositToStoragePayload struct {
	CharacterId   uint32    `json:"characterId"`   // CharacterId initiating the deposit
	AccountId     uint32    `json:"accountId"`     // AccountId that owns the storage
	WorldId       byte      `json:"worldId"`       // WorldId for the storage (storage is world-scoped)
	Slot          int16     `json:"slot"`          // Target slot in storage
	TemplateId    uint32    `json:"templateId"`    // Item template ID
	ReferenceId   uint32    `json:"referenceId"`   // Reference ID for the item data (external service ID)
	ReferenceType string    `json:"referenceType"` // Type of reference: "EQUIPABLE", "CONSUMABLE", "SETUP", "ETC", "CASH"
	Expiration    time.Time `json:"expiration"`    // Item expiration time
	Quantity      uint32    `json:"quantity"`      // Quantity (for stackables)
	OwnerId       uint32    `json:"ownerId"`       // Owner ID (for stackables)
	Flag          uint16    `json:"flag"`          // Item flag (for stackables)
}

// TransferToStoragePayload is the high-level payload for transferring an asset from character to storage
// This step is expanded by saga-orchestrator
type TransferToStoragePayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	WorldId             byte      `json:"worldId"`             // World ID
	AccountId           uint32    `json:"accountId"`           // Account ID (storage owner)
	SourceSlot          int16     `json:"sourceSlot"`          // Slot in character inventory
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type
	Quantity            uint32    `json:"quantity"`            // Quantity to transfer (0 = all)
}

// WithdrawFromStoragePayload is the high-level payload for withdrawing an asset from storage to character
// This step is expanded by saga-orchestrator
type WithdrawFromStoragePayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32    `json:"characterId"`   // Character receiving the item
	WorldId       byte      `json:"worldId"`       // World ID
	AccountId     uint32    `json:"accountId"`     // Account ID (storage owner)
	SourceSlot    int16     `json:"sourceSlot"`    // Slot in storage
	InventoryType byte      `json:"inventoryType"` // Target character inventory type
	Quantity      uint32    `json:"quantity"`      // Quantity to withdraw (0 = all)
}

// TransferToCashShopPayload is the high-level payload for transferring an asset from character to cash shop
// This step is expanded by saga-orchestrator into accept_to_cash_shop + release_from_character
type TransferToCashShopPayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	AccountId           uint32    `json:"accountId"`           // Account ID (cash shop owner)
	CashId              uint64    `json:"cashId"`              // Cash serial number of the item to transfer
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type (equip, use, etc.)
	CompartmentType     byte      `json:"compartmentType"`     // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
}

// WithdrawFromCashShopPayload is the high-level payload for withdrawing an asset from cash shop to character
// This step is expanded by saga-orchestrator into accept_to_character + release_from_cash_shop
type WithdrawFromCashShopPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32    `json:"characterId"`     // Character receiving the item
	AccountId       uint32    `json:"accountId"`       // Account ID (cash shop owner)
	CashId          uint64    `json:"cashId"`          // Cash serial number of the item to withdraw
	CompartmentType byte      `json:"compartmentType"` // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
	InventoryType   byte      `json:"inventoryType"`   // Target character inventory type
}
