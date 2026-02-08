package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeSpawnFromCharacter = "SPAWN_FROM_CHARACTER"
	CommandTypeCancelReservation  = "CANCEL_RESERVATION"
	CommandTypeRequestPickUp      = "REQUEST_PICK_UP"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// EquipmentData carries inline equipment statistics for drops
type EquipmentData struct {
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	Hp            uint16 `json:"hp"`
	Mp            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
}

type SpawnFromCharacterCommandBody struct {
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Mesos      uint32 `json:"mesos"`
	DropType   byte   `json:"dropType"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	OwnerId    uint32 `json:"ownerId"`
	DropperId  uint32 `json:"dropperId"`
	DropperX   int16  `json:"dropperX"`
	DropperY   int16  `json:"dropperY"`
	PlayerDrop bool   `json:"playerDrop"`
	EquipmentData
}

type CancelReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}

type RequestPickUpCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}

const (
	EnvEventTopicDropStatus = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeReserved = "RESERVED"
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

type ReservedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
	EquipmentData
}
