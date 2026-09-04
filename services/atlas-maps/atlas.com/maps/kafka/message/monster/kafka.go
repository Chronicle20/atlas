package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicMonsterStatus topic.Token = "EVENT_TOPIC_MONSTER_STATUS"
	EnvCommandTopic            topic.Token = "COMMAND_TOPIC_MONSTER"
)

const (
	CommandTypeDestroyField = "DESTROY_FIELD"
)

const (
	EventMonsterStatusKilled = "KILLED"
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

type StatusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

// FieldCommand is a field-scoped command on COMMAND_TOPIC_MONSTER. The field
// names and types mirror atlas-monsters' fieldCommand
// (services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go)
// field-for-field. Note there is deliberately NO transactionId: the consumer's
// envelope has none, and an extra key would be silently dropped rather than
// rejected. Any drift here fails open, which is why map/producer_test.go pins
// the emitted key set exactly.
type FieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// DestroyFieldBody matches atlas-monsters' destroyFieldCommandBody: empty.
type DestroyFieldBody struct{}
