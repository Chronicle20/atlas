package guild

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
	StatusEventTypeTitlesUpdated       = "TITLES_UPDATED"
	StatusEventTypeError               = "ERROR"
)

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

type statusEventCreatedBody struct {
}

type statusEventDisbandedBody struct {
	Members []uint32 `json:"members"`
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

type statusEventMemberTitleUpdatedBody struct {
	CharacterId uint32 `json:"characterId"`
	Title       byte   `json:"title"`
}

type statusEventMemberLeftBody struct {
	CharacterId uint32 `json:"characterId"`
	Force       bool   `json:"force"`
}

type statusEventMemberJoinedBody struct {
	CharacterId   uint32 `json:"characterId"`
	Name          string `json:"name"`
	JobId         uint16 `json:"jobId"`
	Level         byte   `json:"level"`
	Title         byte   `json:"title"`
	Online        bool   `json:"online"`
	AllianceTitle byte   `json:"allianceTitle"`
}

type statusEventNoticeUpdatedBody struct {
	Notice string `json:"notice"`
}

type statusEventTitlesUpdatedBody struct {
	GuildId uint32   `json:"guildId"`
	Titles  []string `json:"titles"`
}

type statusEventErrorBody struct {
	ActorId uint32 `json:"actorId"`
	Error   string `json:"error"`
}
