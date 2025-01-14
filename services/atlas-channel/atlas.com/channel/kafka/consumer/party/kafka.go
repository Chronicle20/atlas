package party

const (
	EnvEventStatusTopic         = "EVENT_TOPIC_PARTY_STATUS"
	EventPartyStatusTypeCreated = "CREATED"
	EventPartyStatusTypeJoined  = "JOINED"
	EventPartyStatusTypeLeft    = "LEFT"
	EventPartyStatusTypeExpel   = "EXPEL"
	EventPartyStatusTypeDisband = "DISBAND"
)

type statusEvent[E any] struct {
	WorldId byte   `json:"worldId"`
	PartyId uint32 `json:"partyId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createdEventBody struct {
}

type joinedEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type leftEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type expelEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type disbandEventBody struct {
	CharacterId uint32   `json:"characterId"`
	Members     []uint32 `json:"members"`
}
