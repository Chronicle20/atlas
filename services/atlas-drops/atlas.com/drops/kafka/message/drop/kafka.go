package drop

import "time"

// Event topic and type constants
const (
	EnvEventTopicDropStatus           = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeCreated            = "CREATED"
	StatusEventTypeExpired            = "EXPIRED"
	StatusEventTypePickedUp           = "PICKED_UP"
	StatusEventTypeReserved           = "RESERVED"
	StatusEventTypeReservationFailure = "RESERVATION_FAILURE"
)

// Command topic and type constants
const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeSpawn              = "SPAWN"
	CommandTypeSpawnFromCharacter = "SPAWN_FROM_CHARACTER"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
	CommandTypeCancelReservation  = "CANCEL_RESERVATION"
	CommandTypeRequestPickUp      = "REQUEST_PICK_UP"
)

// StatusEvent is the generic event structure for drop status events
type StatusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	DropId    uint32 `json:"dropId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// Command is the generic command structure for drop commands
type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// StatusEventCreatedBody is the body for CREATED status events
type StatusEventCreatedBody struct {
	ItemId          uint32    `json:"itemId"`
	Quantity        uint32    `json:"quantity"`
	Meso            uint32    `json:"meso"`
	Type            byte      `json:"type"`
	X               int16     `json:"x"`
	Y               int16     `json:"y"`
	OwnerId         uint32    `json:"ownerId"`
	OwnerPartyId    uint32    `json:"ownerPartyId"`
	DropTime        time.Time `json:"dropTime"`
	DropperUniqueId uint32    `json:"dropperUniqueId"`
	DropperX        int16     `json:"dropperX"`
	DropperY        int16     `json:"dropperY"`
	PlayerDrop      bool      `json:"playerDrop"`
}

// StatusEventExpiredBody is the body for EXPIRED status events
type StatusEventExpiredBody struct {
}

// StatusEventPickedUpBody is the body for PICKED_UP status events
type StatusEventPickedUpBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
	PetSlot     int8   `json:"petSlot"`
}

// StatusEventReservedBody is the body for RESERVED status events
type StatusEventReservedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
}

// StatusEventReservationFailureBody is the body for RESERVATION_FAILURE status events
type StatusEventReservationFailureBody struct {
	CharacterId uint32 `json:"characterId"`
}

// CommandSpawnBody is the body for SPAWN commands
type CommandSpawnBody struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Mesos        uint32 `json:"mesos"`
	DropType     byte   `json:"dropType"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	OwnerId      uint32 `json:"ownerId"`
	OwnerPartyId uint32 `json:"ownerPartyId"`
	DropperId    uint32 `json:"dropperId"`
	DropperX     int16  `json:"dropperX"`
	DropperY     int16  `json:"dropperY"`
	PlayerDrop   bool   `json:"playerDrop"`
	Mod          byte   `json:"mod"`
}

// CommandSpawnFromCharacterBody is the body for SPAWN_FROM_CHARACTER commands
type CommandSpawnFromCharacterBody struct {
	ItemId       uint32 `json:"itemId"`
	EquipmentId  uint32 `json:"equipmentId"`
	Quantity     uint32 `json:"quantity"`
	Mesos        uint32 `json:"mesos"`
	DropType     byte   `json:"dropType"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	OwnerId      uint32 `json:"ownerId"`
	OwnerPartyId uint32 `json:"ownerPartyId"`
	DropperId    uint32 `json:"dropperId"`
	DropperX     int16  `json:"dropperX"`
	DropperY     int16  `json:"dropperY"`
	PlayerDrop   bool   `json:"playerDrop"`
	Mod          byte   `json:"mod"`
}

// CommandRequestReservationBody is the body for REQUEST_RESERVATION commands
type CommandRequestReservationBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
	CharacterX  int16  `json:"characterX"`
	CharacterY  int16  `json:"characterY"`
	PetSlot     int8   `json:"petSlot"`
}

// CommandCancelReservationBody is the body for CANCEL_RESERVATION commands
type CommandCancelReservationBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}

// CommandRequestPickUpBody is the body for REQUEST_PICK_UP commands
type CommandRequestPickUpBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}