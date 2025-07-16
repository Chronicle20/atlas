package compartment

import (
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
	CommandRecharge          = "RECHARGE"
	CommandMerge             = "MERGE"
	CommandSort              = "SORT"
	CommandAccept            = "ACCEPT"
	CommandRelease           = "RELEASE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Source    int16  `json:"source"`
	Quantity  int16  `json:"quantity"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
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
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
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

type RechargeCommandBody struct {
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}

type MergeCommandBody struct {
}

type SortCommandBody struct {
}

type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ReferenceId   uint32    `json:"referenceId"`
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
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
	StatusEventTypeAccepted             = "ACCEPTED"
	StatusEventTypeReleased             = "RELEASED"
	StatusEventTypeEquipped             = "EQUIPPED"
	StatusEventTypeUnequipped           = "UNEQUIPPED"
	StatusEventTypeError                = "ERROR"

	AcceptCommandFailed  = "ACCEPT_COMMAND_FAILED"
	ReleaseCommandFailed = "RELEASE_COMMAND_FAILED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
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

type MergeAndSortCompleteEventBody struct {
	Type byte `json:"type"`
}

type MergeCompleteEventBody struct {
	Type byte `json:"type"`
}

type SortCompleteEventBody struct {
	Type byte `json:"type"`
}

type AcceptedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

type ReleasedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

type ErrorEventBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}

type EquippedEventBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type UnequippedEventBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}
