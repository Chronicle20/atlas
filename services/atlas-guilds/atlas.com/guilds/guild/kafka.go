package guild

const (
	EnvStatusEventTopic             = "EVENT_TOPIC_GUILD_STATUS"
	StatusEventTypeCreated          = "CREATED"
	StatusEventTypeEmblemUpdated    = "EMBLEM_UPDATED"
	StatusEventTypeRequestAgreement = "REQUEST_AGREEMENT"
	StatusEventTypeError            = "ERROR"
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

type statusEventEmblemUpdatedBody struct {
	Logo                uint16 `json:"logo"`
	LogoColor           byte   `json:"logoColor"`
	LogoBackground      uint16 `json:"logoBackground"`
	LogoBackgroundColor byte   `json:"logoBackgroundColor"`
}

type statusEventErrorBody struct {
	ActorId uint32 `json:"actorId"`
	Error   string `json:"error"`
}
