package compartment

import (
	"atlas-inventory/kafka/message/asset"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
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
	CommandExpire            = "EXPIRE"
	CommandModifyEquipment   = "MODIFY_EQUIPMENT"
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
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
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
	Slot      int16  `json:"slot"`
	Quantity  uint32 `json:"quantity"`
	RemoveAll bool   `json:"removeAll"` // If true, remove all instances of the item regardless of Quantity
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
	TemplateId    uint32    `json:"templateId"`
	asset.AssetData
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	Quantity      uint32    `json:"quantity"` // Quantity to release (0 = all)
}

// ExpireCommandBody contains the data for expiring an item in the inventory
type ExpireCommandBody struct {
	AssetId        uint32 `json:"assetId"`
	TemplateId     uint32 `json:"templateId"`
	Slot           int16  `json:"slot"`
	ReplaceItemId  uint32 `json:"replaceItemId"`
	ReplaceMessage string `json:"replaceMessage"`
}

// ModifyEquipmentCommandBody contains the data for modifying equipment stats
type ModifyEquipmentCommandBody struct {
	AssetId        uint32    `json:"assetId"`
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
	Slots     uint16 `json:"slots"`
	Flag      uint16 `json:"flag"`
	LevelType byte   `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
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
