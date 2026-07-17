package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply  = "APPLY"
	CommandTypeCancel = "CANCEL"
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
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type CancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
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
