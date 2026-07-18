// Package characterstatus mirrors the atlas-character status events this
// service consumes (source of truth:
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go).
// Only the consumed types/fields are mirrored; unknown event types on the
// topic are ignored by the handlers' type guards.
package characterstatus

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
	StatusEventTypeStatChanged    = "STAT_CHANGED"
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

// StatusEventChannelChangedBody mirrors the producer's
// ChangeChannelEventLoginBody (the upstream name is historical; the JSON
// shape is the contract).
type StatusEventChannelChangedBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

// StatusEventStatChangedBody: Values is populated only for level-up/job flows
// and never carries current HP (verified, design §2) — HP is read via REST in
// the ticker's re-evaluation, so it is not mirrored here.
type StatusEventStatChangedBody struct {
	ChannelId       channel.Id  `json:"channelId"`
	ExclRequestSent bool        `json:"exclRequestSent"`
	Updates         []stat.Type `json:"updates"`
}
