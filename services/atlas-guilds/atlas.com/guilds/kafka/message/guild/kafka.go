package guild

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic                    = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestCreate           = "REQUEST_CREATE"
	CommandTypeRequestInvite           = "REQUEST_INVITE"
	CommandTypeRequestDisband          = "REQUEST_DISBAND"
	CommandTypeRequestCapacityIncrease = "REQUEST_CAPACITY_INCREASE"
	CommandTypeCreationAgreement       = "CREATION_AGREEMENT"
	CommandTypeChangeEmblem            = "CHANGE_EMBLEM"
	CommandTypeChangeNotice            = "CHANGE_NOTICE"
	CommandTypeChangeTitles            = "CHANGE_TITLES"
	CommandTypeChangeMemberTitle       = "CHANGE_MEMBER_TITLE"
	CommandTypeLeave                   = "LEAVE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestCreateBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Name      string `json:"name"`
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

type RequestDisbandBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type RequestCapacityIncreaseBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

const (
	EnvStatusEventTopic                = "EVENT_TOPIC_GUILD_STATUS"
	StatusEventTypeCreated             = "CREATED"
	StatusEventTypeDisbanded           = "DISBANDED"
	StatusEventTypeEmblemUpdated       = "EMBLEM_UPDATED"
	StatusEventTypeRequestAgreement    = "REQUEST_AGREEMENT"
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
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	GuildId       uint32    `json:"guildId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventRequestAgreementBody struct {
	ActorId      uint32 `json:"actorId"`
	ProposedName string `json:"proposedName"`
}

type StatusEventCreatedBody struct {
}

type StatusEventDisbandedBody struct {
	Members []uint32 `json:"members"`
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
