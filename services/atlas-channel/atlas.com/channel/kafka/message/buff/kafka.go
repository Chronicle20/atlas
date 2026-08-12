package buff

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
	CommandTypeCancelByTypes   = "CANCEL_BY_TYPES"
	CommandTypeUpdateStatValue = "UPDATE_STAT_VALUE"
	// CommandTypeExpire asks atlas-buffs to re-evaluate ONE character's buffs
	// and announce whatever has genuinely lapsed. Emitted by the CANCEL_DEBUFF
	// handler. Named EXPIRE rather than RECONCILE because the server does not
	// diff against anything the client claims — the packet carries no payload;
	// it prunes against server-side expiresAt and announces the result.
	// (task-190 FR-2.6.1)
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
	// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
	// NoExpiry marks an explicitly non-expiring buff (task-167 FR-2). When set,
	// Duration MUST be 0; the consumer rejects the command otherwise.
	NoExpiry bool `json:"noExpiry,omitempty"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type CancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
}

type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}

// ExpireCommandBody is deliberately empty: CANCEL_DEBUFF carries no payload,
// so there is nothing for the client to name. The worst a client can assert is
// "please re-check me", and atlas-buffs answers only with buffs that have
// genuinely lapsed. (task-190 FR-2.2 / NFR-2)
type ExpireCommandBody struct{}

// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). Owned by atlas-buffs; this is the
// channel-side mirror. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
	// CreateIfMissing turns INCREMENT into an accumulator upsert: with no buff
	// for SourceId, atlas-buffs creates one with NoExpiry carrying a single
	// StatType change of min(Amount, Cap) and emits APPLIED. Opt-in.
	// (task-216 design.md §4.2)
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

// StatUpdatedStatusEventBody signals a stat value change on an EXISTING buff.
// CreatedAt/ExpiresAt are the buff's original timestamps — the give writers
// encode duration as expiresAt − now, so re-broadcast carries the remaining
// duration and never extends the buff.
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
)

// BerserkStatusEventBody mirrors atlas-buffs' berserk broadcast tick
// (task-154). One event per 3s tick per tracked Dark Knight; Active=false
// ticks clear the aura and keep late-joining observers consistent. ChannelId
// enables the precise sc.Is(tenant, world, channel) guard.
type BerserkStatusEventBody struct {
	TransactionId  uuid.UUID  `json:"transactionId"`
	ChannelId      channel.Id `json:"channelId"`
	SkillId        uint32     `json:"skillId"`
	CharacterLevel byte       `json:"characterLevel"`
	SkillLevel     byte       `json:"skillLevel"`
	Active         bool       `json:"active"`
}
