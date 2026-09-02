// Package buff mirrors the atlas-buffs character-buff commands this service
// PRODUCES (source of truth:
// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go).
// Only the produced type/fields are mirrored, and — unlike a consumer
// mirror — they are exported: this package is the producer side of the
// contract, so events/anniversary needs to construct these values from
// another package.
//
// Anniversary sends CANCEL_BY_CORRELATION (only the zero value for
// WorldId/ChannelId/MapId/Instance/CharacterId — the atlas-buffs consumer
// ignores all of them for this command type, sweeping the whole tenant by
// CorrelationId instead, see the CommandTypeCancelByCorrelation doc comment
// in the source of truth) and APPLY (targeted at the logging-in character,
// task-231 Task 34).
package buff

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_CHARACTER_BUFF"
)

const (

	// CommandTypeApply grants a buff to a single character (FR-A7). Anniversary
	// sends this at login for whichever occurrence(s) are active.
	CommandTypeApply = "APPLY"

	// CommandTypeCancelByCorrelation sweeps the WHOLE TENANT and cancels
	// every buff carrying a given CorrelationId, regardless of which
	// character or world holds it. ONE command rather than one per affected
	// character, so an event occurrence's completion cost does not scale
	// with the online population (FR-A15).
	CommandTypeCancelByCorrelation = "CANCEL_BY_CORRELATION"
)

// Command is the envelope every character-buff command rides in.
type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

// CancelByCorrelationCommandBody names the occurrence (or other correlated
// grant) whose buffs should be swept tenant-wide. CorrelationId is opaque to
// atlas-buffs — it only echoes what buff.Model.CorrelationId() stored.
type CancelByCorrelationCommandBody struct {
	CorrelationId string `json:"correlationId"`
}

// ApplyCommandBody mirrors atlas-buffs
// kafka/message/character/kafka.go's ApplyCommandBody. Only the fields
// Anniversary needs are set: FromId/Level/Accumulate stay zero value.
type ApplyCommandBody struct {
	FromId   uint32 `json:"fromId"`
	SourceId int32  `json:"sourceId"`
	Level    byte   `json:"level"`
	// Duration is MILLISECONDS (contract owner: atlas-buffs). Anniversary
	// always sends 0 alongside NoExpiry: true — the occurrence, not a
	// duration, is the authoritative fact that Anniversary is happening
	// (FR-A5); the atlas-buffs consumer rejects NoExpiry with a nonzero
	// duration.
	Duration   int32        `json:"duration"`
	Changes    []StatChange `json:"changes"`
	Accumulate bool         `json:"accumulate,omitempty"`
	// NoExpiry marks an explicitly non-expiring buff. Anniversary always
	// sets this true — see the Duration doc comment above.
	NoExpiry bool `json:"noExpiry,omitempty"`
	// CorrelationId identifies what granted this buff, for cancel-by-correlation
	// (FR-A12). Anniversary sets this to the granting occurrence's
	// Id().String() — the exact string its own completion path cancels by
	// (see cancelByCorrelationCommandProvider in handler.go).
	CorrelationId string `json:"correlationId,omitempty"`
}

// StatChange mirrors atlas-buffs kafka/message/character/kafka.go's
// StatChange.
type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}
