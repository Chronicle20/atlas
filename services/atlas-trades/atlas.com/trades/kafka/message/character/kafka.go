// Package character carries the EVENT_TOPIC_CHARACTER_STATUS envelopes this
// service consumes to tear a character out of its trade room. Mirrors the
// PRODUCERS of those events:
//
//   - LOGIN / LOGOUT — services/atlas-character/atlas.com/character/kafka/message/character/kafka.go
//   - MAP_CHANGED / CHANNEL_CHANGED — services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go
//     (emitted by services/atlas-maps/atlas.com/maps/kafka/producer/character.go;
//     atlas-character declares neither type)
//
// Struct names, field names and json tags must match those files exactly. Only
// the events this service reacts to are carried over; CREATED and DELETED are
// omitted.
package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin          = "LOGIN"
	EventCharacterStatusTypeLogout         = "LOGOUT"
	EventCharacterStatusTypeChannelChanged = "CHANNEL_CHANGED"
	EventCharacterStatusTypeMapChanged     = "MAP_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventLoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type StatusEventLogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

// StatusEventMapChangedBody carries the full atlas-maps MAP_CHANGED body,
// including the UseTargetPosition/TargetX/TargetY triple that
// MapChangedStatusProvider populates. atlas-trades only needs the map
// transition itself, but a narrowed copy is the drift class these mirrors
// exist to prevent.
type StatusEventMapChangedBody struct {
	ChannelId         channel.Id `json:"channelId"`
	OldMapId          _map.Id    `json:"oldMapId"`
	OldInstance       uuid.UUID  `json:"oldInstance"`
	TargetMapId       _map.Id    `json:"targetMapId"`
	TargetInstance    uuid.UUID  `json:"targetInstance"`
	TargetPortalId    uint32     `json:"targetPortalId"`
	UseTargetPosition bool       `json:"useTargetPosition"`
	TargetX           int16      `json:"targetX"`
	TargetY           int16      `json:"targetY"`
}

type ChangeChannelEventLoginBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}
