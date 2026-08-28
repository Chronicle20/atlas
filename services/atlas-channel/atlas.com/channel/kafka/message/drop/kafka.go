package drop

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic               topic.Token = "COMMAND_TOPIC_DROP"
	CommandTypeRequestReservation             = "REQUEST_RESERVATION"
	CommandTypeSpawn                          = "SPAWN"
	CommandTypeConsume                        = "CONSUME"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type RequestReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
	PartyId     uint32 `json:"partyId"`
	CharacterX  int16  `json:"characterX"`
	CharacterY  int16  `json:"characterY"`
	PetSlot     int8   `json:"petSlot"`
}

// ConsumeCommandBody is the body for CONSUME commands (matches atlas-drops'
// CommandConsumeBody). Consume removes the drop without crediting anyone —
// used to destroy Meso Explosion's detonated meso drops (task-150).
type ConsumeCommandBody struct {
	DropId uint32 `json:"dropId"`
}

// SpawnCommandBody mirrors atlas-drops' CommandSpawnBody field-for-field
// (minus EquipmentData, which is only read for equip item ids and
// zero-fills on decode when absent).
type SpawnCommandBody struct {
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

const (
	EnvEventTopicDropStatus    topic.Token = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeCreated                 = "CREATED"
	StatusEventTypeExpired                 = "EXPIRED"
	StatusEventTypePickedUp                = "PICKED_UP"
	StatusEventTypeConsumed                = "CONSUMED"
	StatusEventTypeMesoAwarded             = "MESO_AWARDED"
)

type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	DropId    uint32     `json:"dropId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreatedStatusEventBody struct {
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
	Mod             byte      `json:"mod"`
}

type ExpiredStatusEventBody struct{}

type ConsumedStatusEventBody struct{}

type PickedUpStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
	PetSlot     int8   `json:"petSlot"`
}

// MesoAwardedStatusEventBody mirrors atlas-drops' StatusEventMesoAwardedBody.
// One event per recipient of a split meso drop; exactly one carries
// Picker: true, and only that one completes the pickup.
type MesoAwardedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	Amount      uint32 `json:"amount"`
	Picker      bool   `json:"picker"`
}
