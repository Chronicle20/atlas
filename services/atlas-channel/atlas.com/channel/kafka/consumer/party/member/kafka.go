package member

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_PARTY_MEMBER_STATUS"
	EventPartyStatusTypeLogin  = "LOGIN"
	EventPartyStatusTypeLogout = "LOGOUT"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	PartyId     uint32 `json:"partyId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type loginEventBody struct {
}

type logoutEventBody struct {
}
