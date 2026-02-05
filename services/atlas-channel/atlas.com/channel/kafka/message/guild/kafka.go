package guild

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic              = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName       = "REQUEST_NAME"
	CommandTypeRequestEmblem     = "REQUEST_EMBLEM"
	CommandTypeRequestCreate     = "REQUEST_CREATE"
	CommandTypeRequestInvite     = "REQUEST_INVITE"
	CommandTypeCreationAgreement = "CREATION_AGREEMENT"
	CommandTypeChangeEmblem      = "CHANGE_EMBLEM"
	CommandTypeChangeNotice      = "CHANGE_NOTICE"
	CommandTypeChangeTitles      = "CHANGE_TITLES"
	CommandTypeChangeMemberTitle = "CHANGE_MEMBER_TITLE"
	CommandTypeLeave             = "LEAVE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestNameBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestEmblemBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestCreateBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Name      string     `json:"name"`
}

type CreationAgreementBody struct {
	Agreed bool `json:"agreed"`
}

type ChangeEmblemBody struct {
	GuildId             uint32 `json:"guildId"`
	Logo                uint16 `json:"logo"`
	LogoColor           byte   `json:"logoColor"`
	LogoBackground      uint16 `json:"logoBackground"`
	LogoBackgroundColor byte   `json:"logoBackgroundColor"`
}

type ChangeNoticeBody struct {
	GuildId uint32 `json:"guildId"`
	Notice  string `json:"notice"`
}

type LeaveBody struct {
	GuildId uint32 `json:"guildId"`
	Force   bool   `json:"force"`
}

type RequestInviteBody struct {
	GuildId  uint32 `json:"guildId"`
	TargetId uint32 `json:"targetId"`
}

type ChangeTitlesBody struct {
	GuildId uint32   `json:"guildId"`
	Titles  []string `json:"titles"`
}

type ChangeMemberTitleBody struct {
	GuildId  uint32 `json:"guildId"`
	TargetId uint32 `json:"targetId"`
	Title    byte   `json:"title"`
}

const (
	EnvStatusEventTopic                = "EVENT_TOPIC_GUILD_STATUS"
	StatusEventTypeRequestAgreement    = "REQUEST_AGREEMENT"
	StatusEventTypeCreated             = "CREATED"
	StatusEventTypeDisbanded           = "DISBANDED"
	StatusEventTypeEmblemUpdated       = "EMBLEM_UPDATED"
	StatusEventTypeMemberStatusUpdated = "MEMBER_STATUS_UPDATED"
	StatusEventTypeMemberTitleUpdated  = "MEMBER_TITLE_UPDATED"
	StatusEventTypeMemberLeft          = "MEMBER_LEFT"
	StatusEventTypeMemberJoined        = "MEMBER_JOINED"
	StatusEventTypeNoticeUpdated       = "NOTICE_UPDATED"
	StatusEventTypeCapacityUpdated     = "CAPACITY_UPDATED"
	StatusEventTypeTitlesUpdated       = "TITLES_UPDATED"
	StatusEventTypeError               = "ERROR"
)

type StatusEvent[E any] struct {
	WorldId world.Id `json:"worldId"`
	GuildId uint32   `json:"guildId"`
	Type    string   `json:"type"`
	Body    E        `json:"body"`
}

type StatusEventCreatedBody struct {
}

type StatusEventDisbandedBody struct {
	Members []uint32 `json:"members"`
}

type StatusEventRequestAgreementBody struct {
	ActorId      uint32 `json:"actorId"`
	ProposedName string `json:"proposedName"`
}

type StatusEventEmblemUpdatedBody struct {
	Logo                uint16 `json:"logo"`
	LogoColor           byte   `json:"logoColor"`
	LogoBackground      uint16 `json:"logoBackground"`
	LogoBackgroundColor byte   `json:"logoBackgroundColor"`
}

type StatusEventMemberStatusUpdatedBody struct {
	CharacterId uint32 `json:"characterId"`
	Online      bool   `json:"online"`
}

type StatusEventMemberTitleUpdatedBody struct {
	CharacterId uint32 `json:"characterId"`
	Title       byte   `json:"title"`
}

type StatusEventMemberLeftBody struct {
	CharacterId uint32 `json:"characterId"`
	Force       bool   `json:"force"`
}

type StatusEventMemberJoinedBody struct {
	CharacterId   uint32 `json:"characterId"`
	Name          string `json:"name"`
	JobId         uint16 `json:"jobId"`
	Level         byte   `json:"level"`
	Title         byte   `json:"title"`
	Online        bool   `json:"online"`
	AllianceTitle byte   `json:"allianceTitle"`
}

type StatusEventNoticeUpdatedBody struct {
	Notice string `json:"notice"`
}

type StatusEventCapacityUpdatedBody struct {
	Capacity uint32 `json:"capacity"`
}

type StatusEventTitlesUpdatedBody struct {
	GuildId uint32   `json:"guildId"`
	Titles  []string `json:"titles"`
}

type StatusEventErrorBody struct {
	ActorId uint32 `json:"actorId"`
	Error   string `json:"error"`
}
