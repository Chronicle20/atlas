package compartment

import (
	"time"

	"github.com/google/uuid"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_COMPARTMENT"
	CommandRequestReserve    = "REQUEST_RESERVE"
	CommandConsume           = "CONSUME"
	CommandDestroy           = "DESTROY"
	CommandCancelReservation = "CANCEL_RESERVATION"
	CommandModifyEquipment   = "MODIFY_EQUIPMENT"
	CommandCreateAsset       = "CREATE_ASSET"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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

type CreateAssetCommandBody struct {
	TemplateId      uint32    `json:"templateId"`
	Quantity        uint32    `json:"quantity"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Flag            uint16    `json:"flag"`
	Rechargeable    uint64    `json:"rechargeable"`
	UseAverageStats bool      `json:"useAverageStats,omitempty"`
}

const (
	EnvEventTopicStatus                 = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeReserved             = "RESERVED"
	StatusEventTypeReservationCancelled = "RESERVATION_CANCELLED"

	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"

	CreateAssetTemplateNotFound = "CREATE_ASSET_TEMPLATE_NOT_FOUND"
	CreateAssetInventoryFull    = "CREATE_ASSET_INVENTORY_FULL"
	CreateAssetUnknownError     = "CREATE_ASSET_UNKNOWN_ERROR"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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

// CreateResultEventBody is a union over the CREATED ({type,capacity}) and
// CREATION_FAILED ({errorCode,message}) event bodies so a single once-handler
// can await either; branch on StatusEvent.Type.
type CreateResultEventBody struct {
	Type      byte   `json:"type"`
	Capacity  uint32 `json:"capacity"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

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
	Slots          uint16    `json:"slots"`
	Flag           uint16    `json:"flag"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}
