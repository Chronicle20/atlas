package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_DROP"
	CommandTypeSpawn = "SPAWN"
)

type command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

func commandFromField[E any](f field.Model, theType string, body E) command[E] {
	return command[E]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      theType,
		Body:      body,
	}
}

type spawnCommandBody struct {
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
