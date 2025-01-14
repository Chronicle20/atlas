package party

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate       = "CREATE"
	CommandPartyJoin         = "JOIN"
	CommandPartyLeave        = "LEAVE"
	CommandPartyChangeLeader = "CHANGE_LEADER"

	EnvEventStatusTopic         = "EVENT_TOPIC_PARTY_STATUS"
	EventPartyStatusTypeCreated = "CREATED"
	EventPartyStatusTypeJoined  = "JOINED"
	EventPartyStatusTypeLeft    = "LEFT"
	EventPartyStatusTypeExpel   = "EXPEL"
	EventPartyStatusTypeDisband = "DISBAND"
)

type commandEvent[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type createCommandBody struct {
	LeaderId uint32 `json:"leaderId"`
}

type joinCommandBody struct {
	PartyId     uint32 `json:"partyId"`
	CharacterId uint32 `json:"characterId"`
}

type leaveCommandBody struct {
	PartyId     uint32 `json:"partyId"`
	CharacterId uint32 `json:"characterId"`
	Force       bool   `json:"force"`
}

type changeLeaderBody struct {
	LeaderId uint32 `json:"leaderId"`
	PartyId  uint32 `json:"partyId"`
}

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
