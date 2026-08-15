package character

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply           = "APPLY"
	CommandTypeCancel          = "CANCEL"
	CommandTypeCancelAll       = "CANCEL_ALL"
	CommandTypeCancelByTypes   = "CANCEL_BY_TYPES"
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"
	// CommandTypeExpire asks for ONE character's buffs to be re-evaluated and
	// whatever has genuinely lapsed announced. Emitted by atlas-channel's
	// CANCEL_DEBUFF handler (task-190 FR-2.6.1). Named EXPIRE rather than
	// RECONCILE because there is no two-way diff — the client's packet carries
	// no payload; this prunes against server-side expiresAt.
	CommandTypeExpire = "EXPIRE"

	// Operations for UPDATE_STAT_VALUE. INCREMENT adds Amount clamped to Cap;
	// SET replaces the stat amount outright (finisher consume = SET 1).
	StatOperationIncrement = "INCREMENT"
	StatOperationSet       = "SET"
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
	FromId   uint32 `json:"fromId"`
	SourceId int32  `json:"sourceId"`
	Level    byte   `json:"level"`
	// Duration is MILLISECONDS. This is the single authoritative statement of
	// the COMMAND_TOPIC_CHARACTER_BUFF duration unit: atlas-buffs is the
	// consumer that defines it (buff.NewBuff computes
	// expiresAt = now + duration*time.Millisecond), so the unit is its property
	// to declare. Every producer's local copy of this struct carries a one-line
	// pointer back here rather than restating the rule — three separate
	// commits (11e07dfa7, 197324e40, 88d270bf1) flipped it in prose alone.
	// tools/buff-duration-guard.sh fails CI on a seconds-valued emitter.
	// (task-190 FR-3.1)
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
	// Accumulate, when true, stores each change as its own independently-timed
	// buff under the same sourceId (per-stat keying) instead of replacing the
	// whole sourceId buff. Used by the Beholder Hex sweep so its buffs accumulate
	// one-at-a-time (original-GMS behavior). Default false preserves the standard
	// replace-by-sourceId semantics for every other producer.
	Accumulate bool `json:"accumulate,omitempty"`
	// NoExpiry marks an explicitly non-expiring buff (task-167 FR-2). When set,
	// Duration MUST be 0; the consumer rejects the command otherwise.
	NoExpiry bool `json:"noExpiry,omitempty"`
	// CorrelationId identifies what granted this buff, for cancel-by-correlation
	// (FR-A12). Opaque to atlas-buffs. Optional — omitting it leaves every
	// existing producer's bytes unchanged.
	CorrelationId string `json:"correlationId,omitempty"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type CancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
}

type CancelAllCommandBody struct{}

type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}

// ExpireCommandBody is deliberately empty: CANCEL_DEBUFF carries no payload, so
// a client cannot name anything. Honoring it unconditionally is provably safe —
// the worst assertion is "please re-check me", and only genuinely lapsed buffs
// are announced. Amplification is bounded upstream by atlas-channel's
// per-character throttle. (task-190 FR-2.2 / NFR-2)
type ExpireCommandBody struct{}

// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). The body is stat-generic; task-142
// uses it for COMBO orb bookkeeping. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
	// CreateIfMissing turns INCREMENT into an accumulator upsert: with no
	// buff for SourceId, one is created with NoExpiry carrying a single
	// StatType change of min(Amount, Cap), and APPLIED (not STAT_UPDATED) is
	// emitted. Opt-in — omitted/false leaves every existing producer's
	// behaviour byte-identical. (task-216 design.md §4.2)
	CreateIfMissing bool `json:"createIfMissing,omitempty"`
	// Level is the source skill level stamped on a buff created by
	// CreateIfMissing. Ignored otherwise.
	Level byte `json:"level,omitempty"`
}

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
	EventStatusTypeStatUpdated = "STAT_UPDATED"
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
	NoExpiry  bool         `json:"noExpiry,omitempty"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
	NoExpiry  bool         `json:"noExpiry,omitempty"`
}

// StatUpdatedStatusEventBody is emitted when a stat value on an existing buff
// changed (not a new buff — consumers that react to APPLIED as "a buff came
// into existence" must ignore this type). CreatedAt/ExpiresAt are the buff's
// ORIGINAL timestamps so re-broadcast carries the remaining duration.
type StatUpdatedStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

const (
	EventStatusTypeBerserk = "BERSERK"

	// EventStatusTypePeriodicEffect is one visual pulse of a periodic buff
	// effect (task-214). Emitted alongside -- never instead of -- the tick's
	// CHANGE_HP command, and only for periodic-effect rows whose source skill
	// actually has a `special` WZ node to draw (see periodic.Effect.SpecialEffect).
	EventStatusTypePeriodicEffect = "PERIODIC_EFFECT"
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

// PeriodicEffectStatusEventBody is one visual pulse for one periodic tick.
// SkillId is the buff's source skill; the client's SKILL_SPECIAL user effect
// carries nothing else, so no level rides along. ChannelId is in the body for
// the same reason as BerserkStatusEventBody's: this topic's envelope has no
// channel, and atlas-channel needs it for the sc.Is(tenant, world, channel)
// guard. StatType is carried for logging/diagnosis only -- the channel does
// not branch on it.
type PeriodicEffectStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	SkillId   uint32     `json:"skillId"`
	StatType  string     `json:"statType"`
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
