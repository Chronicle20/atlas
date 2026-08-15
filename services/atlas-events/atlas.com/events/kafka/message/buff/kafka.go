// Package buff mirrors the atlas-buffs character-buff commands this service
// PRODUCES (source of truth:
// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go).
// Only the produced type/fields are mirrored, and — unlike a consumer
// mirror — they are exported: this package is the producer side of the
// contract, so events/anniversary needs to construct these values from
// another package.
//
// Anniversary only ever sends CANCEL_BY_CORRELATION, and only ever the zero
// value for WorldId/ChannelId/MapId/Instance/CharacterId: the atlas-buffs
// consumer ignores all of them for this command type — it sweeps the whole
// tenant by CorrelationId (see the CommandTypeCancelByCorrelation doc
// comment in the source of truth).
package buff

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER_BUFF"

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
