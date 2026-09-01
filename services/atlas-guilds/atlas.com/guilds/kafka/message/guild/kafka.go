package guild

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_GUILD"
)

const (
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
	// CommandTypeRejoin re-adds a character to a guild at an explicitly
	// supplied title. It exists for the world-transfer saga's compensation
	// (task-227 FR-4.8): leave_guild_for_transfer emits a FORCED LEAVE, and a
	// guild re-join is not a client-driveable recovery, so the only way to put
	// the player back where they were is a server-issued re-add that restores
	// the exact rank they held. It is deliberately NOT the invite/accept flow
	// (which would require the player to act) and NOT REQUEST_INVITE.
	CommandTypeRejoin = "REJOIN"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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

// RejoinBody carries the guild and the rank the character must be restored to.
// Title is mandatory: re-adding at the default rookie title would silently
// demote a guild officer whose world transfer failed.
type RejoinBody struct {
	GuildId uint32 `json:"guildId"`
	Title   byte   `json:"title"`
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
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestCapacityIncreaseBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

const (
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_GUILD_STATUS"
)

const (
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
	WorldId       world.Id  `json:"worldId"`
	GuildId       uint32    `json:"guildId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventRequestAgreementBody struct {
	ActorId      uint32 `json:"actorId"`
	ProposedName string `json:"proposedName"`
}

type StatusEventCreatedBody struct{}

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
