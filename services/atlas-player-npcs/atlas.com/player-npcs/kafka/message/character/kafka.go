// Package character holds the subset of atlas-character's
// EVENT_TOPIC_CHARACTER_STATUS envelope the LEVEL_CHANGED consumer (Task
// 17) reads -- ported from
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:307-311/265-271.
// Only LEVEL_CHANGED is defined here; this package has no reason to mirror
// atlas-character's full event catalog.
package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicCharacterStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)
const (
	StatusEventTypeLevelChanged = "LEVEL_CHANGED"
)

// StatusEvent mirrors atlas-character's StatusEvent[E] envelope
// (kafka.go:265-271).
type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

// LevelChangedStatusEventBody mirrors atlas-character's
// LevelChangedStatusEventBody (kafka.go:307-311) verbatim: no job, no gm
// flag, no map -- design §8.2's rationale for why this consumer fetches
// the character.
type LevelChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    byte       `json:"amount"`
	Current   byte       `json:"current"`
}
