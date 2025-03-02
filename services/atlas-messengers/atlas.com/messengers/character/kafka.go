package character

const (
	EnvEventMemberStatusTopic            = "EVENT_TOPIC_MESSENGER_MEMBER_STATUS"
	EventMessengerMemberStatusTypeLogin  = "LOGIN"
	EventMessengerMemberStatusTypeLogout = "LOGOUT"
)

type memberStatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	MessengerId uint32 `json:"messengerId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type memberLoginEventBody struct {
}

type memberLogoutEventBody struct {
}
