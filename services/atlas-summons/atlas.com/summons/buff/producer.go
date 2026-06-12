// Package buff declares atlas-summons' local view of the atlas-buffs command
// contract (COMMAND_TOPIC_CHARACTER_BUFF) and the provider used by the Beholder
// aura sweep to apply the Beholder buff to a summon's owner.
//
// Services in this monorepo never import one another, so the envelope and body
// shapes here are re-declared to match the atlas-buffs consumer at
// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
// (Command[E] at lines 20-28, ApplyCommandBody at lines 30-36, StatChange at
// lines 38-41). The JSON tags MUST stay byte-identical to that consumer or the
// buff silently fails.
package buff

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	// EnvCommandTopic names the env var holding the atlas-buffs command topic.
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER_BUFF"

	// CommandTypeApply applies a buff to a character. Mirrors atlas-buffs
	// CommandTypeApply.
	CommandTypeApply = "APPLY"
)

// Command is the atlas-buffs command envelope. Tags match
// buffs/kafka/message/character/kafka.go:20-28.
type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

// ApplyCommandBody mirrors buffs/kafka/message/character/kafka.go:30-36.
type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
}

// StatChange mirrors buffs/kafka/message/character/kafka.go:38-41.
type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

// applyProvider applies a buff to characterId. Used by the Beholder aura sweep to
// re-apply the Beholder buff to the owner (FR-5.x). The key is the character id so
// all of a character's buff commands stay ordered on one partition.
func applyProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []StatChange) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[ApplyCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        CommandTypeApply,
		Body: ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Level:    level,
			Duration: duration,
			Changes:  changes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ApplyProvider is the exported entry point for the Beholder aura sweep.
func ApplyProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []StatChange) model.Provider[[]kafka.Message] {
	return applyProvider(f, characterId, fromId, sourceId, level, duration, changes)
}
