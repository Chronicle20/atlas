package message

import (
	"time"

	"github.com/Chronicle20/atlas-constants/asset"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_STORAGE"
	EnvEventTopic   = "EVENT_TOPIC_STORAGE_STATUS"
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
	CommandTypeExpire          = "EXPIRE"
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

// AssetData carries all inline asset fields for commands and events
type AssetData struct {
	Expiration     time.Time `json:"expiration"`
	Quantity       uint32    `json:"quantity"`
	OwnerId        uint32    `json:"ownerId"`
	Flag           uint16    `json:"flag"`
	Rechargeable   uint64    `json:"rechargeable"`
	Strength       uint16    `json:"strength"`
	Dexterity      uint16    `json:"dexterity"`
	Intelligence   uint16    `json:"intelligence"`
	Luck           uint16    `json:"luck"`
	Hp             uint16    `json:"hp"`
	Mp             uint16    `json:"mp"`
	WeaponAttack   uint16    `json:"weaponAttack"`
	MagicAttack    uint16    `json:"magicAttack"`
	WeaponDefense  uint16    `json:"weaponDefense"`
	MagicDefense   uint16    `json:"magicDefense"`
	Accuracy       uint16    `json:"accuracy"`
	Avoidability   uint16    `json:"avoidability"`
	Hands          uint16    `json:"hands"`
	Speed          uint16    `json:"speed"`
	Jump           uint16    `json:"jump"`
	Slots          uint16    `json:"slots"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	CashId         int64     `json:"cashId,string"`
	CommodityId    uint32    `json:"commodityId"`
	PurchaseBy     uint32    `json:"purchaseBy"`
	PetId          uint32    `json:"petId"`
}

// Command is the base command structure for storage operations
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEvent is the base event structure for storage status updates
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// DepositBody contains the data needed to deposit an item into storage
type DepositBody struct {
	Slot       int16  `json:"slot"`
	TemplateId uint32 `json:"templateId"`
	AssetData
}

// WithdrawBody contains the data needed to withdraw an item from storage
type WithdrawBody struct {
	AssetId  asset.Id       `json:"assetId"`
	Quantity asset.Quantity `json:"quantity,omitempty"`
}

// UpdateMesosBody contains the data needed to update mesos in storage
type UpdateMesosBody struct {
	Mesos     uint32 `json:"mesos"`
	Operation string `json:"operation"` // "SET", "ADD", "SUBTRACT"
}

// DepositRollbackBody contains the data needed to rollback a deposit
type DepositRollbackBody struct {
	AssetId asset.Id `json:"assetId"`
}

// DepositedEventBody contains the data for a deposited event
type DepositedEventBody struct {
	AssetId    asset.Id `json:"assetId"`
	Slot       int16    `json:"slot"`
	TemplateId uint32   `json:"templateId"`
}

// WithdrawnEventBody contains the data for a withdrawn event
type WithdrawnEventBody struct {
	AssetId    asset.Id       `json:"assetId"`
	Slot       int16          `json:"slot"`
	TemplateId uint32         `json:"templateId"`
	Quantity   asset.Quantity `json:"quantity,omitempty"`
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
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	NpcId         uint32     `json:"npcId"`
	AccountId     uint32     `json:"accountId"`
	Type          string     `json:"type"`
}

// CloseStorageCommand is received when a character closes storage
type CloseStorageCommand struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
}

// ProjectionCreatedEventBody contains the data for a projection created event
type ProjectionCreatedEventBody struct {
	CharacterId uint32     `json:"characterId"`
	AccountId   uint32     `json:"accountId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	NpcId       uint32     `json:"npcId"`
}

// ProjectionDestroyedEventBody contains the data for a projection destroyed event
type ProjectionDestroyedEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

// ExpireBody contains the data for expiring an item in storage
type ExpireBody struct {
	CharacterId    uint32   `json:"characterId"`
	AssetId        asset.Id `json:"assetId"`
	TemplateId     uint32   `json:"templateId"`
	InventoryType  int8     `json:"inventoryType"`
	Slot           int16    `json:"slot"`
	ReplaceItemId  uint32   `json:"replaceItemId"`
	ReplaceMessage string   `json:"replaceMessage"`
}

// ExpiredStatusEventBody contains information about an expired item for client notification
type ExpiredStatusEventBody struct {
	IsCash         bool   `json:"isCash"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}
