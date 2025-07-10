package drop

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type RequestReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
	CharacterX  int16  `json:"characterX"`
	CharacterY  int16  `json:"characterY"`
	PetSlot     int8   `json:"petSlot"`
}

const (
	EnvEventTopicDropStatus = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeCreated  = "CREATED"
	StatusEventTypeExpired  = "EXPIRED"
	StatusEventTypePickedUp = "PICKED_UP"
)

type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
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

type ExpiredStatusEventBody struct {
}

type PickedUpStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
	PetSlot     int8   `json:"petSlot"`
}
