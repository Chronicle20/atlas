package guild

const (
	EnvCommandTopic          = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName   = "REQUEST_NAME"
	CommandTypeRequestEmblem = "REQUEST_EMBLEM"

	EnvStatusEventTopic                = "EVENT_TOPIC_GUILD_STATUS"
	StatusEventTypeRequestAgreement    = "REQUEST_AGREEMENT"
	StatusEventTypeEmblemUpdated       = "EMBLEM_UPDATED"
	StatusEventTypeMemberStatusUpdated = "MEMBER_STATUS_UPDATED"
	StatusEventTypeMemberLeft          = "MEMBER_LEFT"
	StatusEventTypeNoticeUpdated       = "NOTICE_UPDATED"
	StatusEventTypeError               = "ERROR"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestNameBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type requestEmblemBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type statusEvent[E any] struct {
	WorldId byte   `json:"worldId"`
	GuildId uint32 `json:"guildId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type statusEventRequestAgreementBody struct {
	ActorId      uint32 `json:"actorId"`
	ProposedName string `json:"proposedName"`
}

type statusEventEmblemUpdatedBody struct {
	Logo                uint16 `json:"logo"`
	LogoColor           byte   `json:"logoColor"`
	LogoBackground      uint16 `json:"logoBackground"`
	LogoBackgroundColor byte   `json:"logoBackgroundColor"`
}

type statusEventMemberStatusUpdatedBody struct {
	CharacterId uint32 `json:"characterId"`
	Online      bool   `json:"online"`
}

type statusEventMemberLeftBody struct {
	CharacterId uint32 `json:"characterId"`
	Force       bool   `json:"force"`
}

type statusEventNoticeUpdatedBody struct {
	Notice string `json:"notice"`
}

type statusEventErrorBody struct {
	ActorId uint32 `json:"actorId"`
	Error   string `json:"error"`
}
