package character

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply         = "APPLY"
	CommandTypeCancel        = "CANCEL"
	CommandTypeCancelAll     = "CANCEL_ALL"
	CommandTypeCancelByTypes = "CANCEL_BY_TYPES"
)

type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
	// Accumulate, when true, stores each change as its own independently-timed
	// buff under the same sourceId (per-stat keying) instead of replacing the
	// whole sourceId buff. Used by the Beholder Hex sweep so its buffs accumulate
	// one-at-a-time (original-GMS behavior). Default false preserves the standard
	// replace-by-sourceId semantics for every other producer.
	Accumulate bool `json:"accumulate,omitempty"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type CancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
}

type CancelAllCommandBody struct {
}

type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type AppliedStatusEventBody struct {
	FromId    uint32       `json:"fromId"`
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

const (
	EventStatusTypeBerserk = "BERSERK"
)

// BerserkStatusEventBody is one broadcast tick of Dark Knight Berserk aura
// state (task-154). Emitted every BroadcastPeriod per tracked Dark Knight
// with the state captured at the last re-evaluation; Active=false ticks are
// intentional — they clear the aura and keep late-joining observers
// consistent. ChannelId rides in the body because this topic's envelope has
// no channel; it lets atlas-channel guard with sc.Is(tenant, world, channel).
type BerserkStatusEventBody struct {
	TransactionId  uuid.UUID  `json:"transactionId"`
	ChannelId      channel.Id `json:"channelId"`
	SkillId        uint32     `json:"skillId"`
	CharacterLevel byte       `json:"characterLevel"`
	SkillLevel     byte       `json:"skillLevel"`
	Active         bool       `json:"active"`
}

const (
	EnvCommandTopicCharacter = "COMMAND_TOPIC_CHARACTER"
	CommandChangeHP          = "CHANGE_HP"
)

type CharacterCommand[E any] struct {
	CharacterId uint32   `json:"characterId"`
	WorldId     world.Id `json:"worldId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type ChangeHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}
