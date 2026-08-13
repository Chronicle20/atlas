package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeJobChanged     = "JOB_CHANGED"
)

// StatusEvent mirrors the atlas-character EVENT_TOPIC_CHARACTER_STATUS envelope
// (services/atlas-character/.../kafka/message/character/kafka.go). Bodies are
// decoded faithfully to avoid Kafka parse errors even where a field is unused.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type LoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type LogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type MapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type ChannelChangedBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

// JobChangedBody is the only status body carrying the job id, and it carries no
// map id — so a job change into the dragon-bearing range must resolve the
// character's field from the character service, not from the event.
type JobChangedBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
}
