package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_MONSTER"
)

const (
	CommandTypeCatch = "CATCH"
)

// Command mirrors atlas-monsters' shared command envelope. MonsterId carries
// the mob's unique (field object) id, matching every sibling command.
type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// CatchCommandBody is deliberately minimal: every handler on the shared monster
// command topic unmarshals every message, so a field whose type disagrees with a
// sibling body logs one spurious error per message.
type CatchCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

const (
	EnvEventTopicCatch topic.Token = "EVENT_TOPIC_MONSTER_CATCH"
)

const (
	EventMonsterCatchResolved = "CATCH_RESOLVED"
)

// Event mirrors atlas-monsters' status-event envelope on the dedicated catch
// topic. This service deliberately does NOT subscribe to
// EVENT_TOPIC_MONSTER_STATUS: that topic carries a DAMAGED event per hit and
// every registered handler unmarshals every message (design §4.2).
type Event[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CatchResolvedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Success     bool   `json:"success"`
	Cause       string `json:"cause"`
}
