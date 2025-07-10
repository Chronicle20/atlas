package compartment

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"time"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_COMPARTMENT"
	CommandEquip             = "EQUIP"
	CommandUnequip           = "UNEQUIP"
	CommandMove              = "MOVE"
	CommandDrop              = "DROP"
	CommandRequestReserve    = "REQUEST_RESERVE"
	CommandConsume           = "CONSUME"
	CommandDestroy           = "DESTROY"
	CommandCancelReservation = "CANCEL_RESERVATION"
	CommandIncreaseCapacity  = "INCREASE_CAPACITY"
	CommandCreateAsset       = "CREATE_ASSET"
	CommandMerge             = "MERGE"
	CommandSort              = "SORT"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type EquipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type UnequipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type MoveCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type DropCommandBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Source    int16      `json:"source"`
	Quantity  int16      `json:"quantity"`
	X         int16      `json:"x"`
	Y         int16      `json:"y"`
}

type RequestReserveCommandBody struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	Items         []ItemBody `json:"items"`
}

type ItemBody struct {
	Source   int16  `json:"source"`
	ItemId   uint32 `json:"itemId"`
	Quantity int16  `json:"quantity"`
}

type ConsumeCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}

type DestroyCommandBody struct {
	Slot int16 `json:"slot"`
}

type CancelReservationCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}

type IncreaseCapacityCommandBody struct {
	Amount uint32 `json:"amount"`
}

type CreateAssetCommandBody struct {
	TemplateId   uint32    `json:"templateId"`
	Quantity     uint32    `json:"quantity"`
	Expiration   time.Time `json:"expiration"`
	OwnerId      uint32    `json:"ownerId"`
	Flag         uint16    `json:"flag"`
	Rechargeable uint64    `json:"rechargeable"`
}

type MergeCommandBody struct {
}

type SortCommandBody struct {
}

type MoveToCommandBody struct {
	Slot           int16  `json:"slot"`
	OtherInventory string `json:"otherInventory"`
}

const (
	EnvCommandTopicCompartmentTransfer = "COMMAND_TOPIC_COMPARTMENT_TRANSFER"
	InventoryTypeCharacter             = "CHARACTER"
	InventoryTypeCashShop              = "CASH_SHOP"
)

type TransferCommand struct {
	TransactionId       uuid.UUID `json:"transactionId"`
	AccountId           uint32    `json:"accountId"`
	CharacterId         uint32    `json:"characterId"`
	AssetId             uint32    `json:"assetId"`
	FromCompartmentId   uuid.UUID `json:"fromCompartmentId"`
	FromCompartmentType byte      `json:"fromCompartmentType"`
	FromInventoryType   string    `json:"fromInventoryType"`
	ToCompartmentId     uuid.UUID `json:"toCompartmentId"`
	ToCompartmentType   byte      `json:"toCompartmentType"`
	ToInventoryType     string    `json:"toInventoryType"`
	ReferenceId         uint32    `json:"referenceId"`
}

const (
	EnvEventTopicStatus                 = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeCreated              = "CREATED"
	StatusEventTypeDeleted              = "DELETED"
	StatusEventTypeCapacityChanged      = "CAPACITY_CHANGED"
	StatusEventTypeReserved             = "RESERVED"
	StatusEventTypeReservationCancelled = "RESERVATION_CANCELLED"
	StatusEventTypeMergeComplete        = "MERGE_COMPLETE"
	StatusEventTypeSortComplete         = "SORT_COMPLETE"
	StatusEventTypeCompleted            = "COMPLETED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Type     byte   `json:"type"`
	Capacity uint32 `json:"capacity"`
}

type DeletedStatusEventBody struct {
}

type CapacityChangedEventBody struct {
	Type     byte   `json:"type"`
	Capacity uint32 `json:"capacity"`
}

type ReservedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ItemId        uint32    `json:"itemId"`
	Slot          int16     `json:"slot"`
	Quantity      uint32    `json:"quantity"`
}
type ReservationCancelledEventBody struct {
	ItemId   uint32 `json:"itemId"`
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}

type MergeCompleteEventBody struct {
	Type byte `json:"type"`
}

type SortCompleteEventBody struct {
	Type byte `json:"type"`
}

const (
	EnvEventTopicCompartmentTransferStatus = "EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS"
)

type TransferStatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// TransferStatusEventCompletedBody represents the body of a COMPLETED status event
type TransferStatusEventCompletedBody struct {
	TransactionId   uuid.UUID `json:"transactionId"`
	AccountId       uint32    `json:"accountId"`
	AssetId         uint32    `json:"assetId"`
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	InventoryType   string    `json:"inventoryType"`
}
