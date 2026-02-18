package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventStatusDamaged      = "DAMAGED"
	EventStatusKilled       = "KILLED"
	EventStatusFriendlyDrop = "FRIENDLY_DROP"
)

type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

func (e StatusEvent[E]) Field() field.Model {
	return field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
}

type DamagedBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ObserverId    uint32        `json:"observerId"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

type KilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type FriendlyDropBody struct {
	ItemCount uint32 `json:"itemCount"`
}
