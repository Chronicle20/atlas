package party

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate       = "CREATE"
	CommandPartyJoin         = "JOIN"
	CommandPartyLeave        = "LEAVE"
	CommandPartyChangeLeader = "CHANGE_LEADER"

	EnvEventStatusTopic              = "EVENT_TOPIC_PARTY_STATUS"
	EventPartyStatusTypeCreated      = "CREATED"
	EventPartyStatusTypeJoined       = "JOINED"
	EventPartyStatusTypeLeft         = "LEFT"
	EventPartyStatusTypeExpel        = "EXPEL"
	EventPartyStatusTypeDisband      = "DISBAND"
	EventPartyStatusTypeChangeLeader = "CHANGE_LEADER"
)

type commandEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createCommandBody struct {
}

type joinCommandBody struct {
	PartyId uint32 `json:"partyId"`
}

type leaveCommandBody struct {
	PartyId uint32 `json:"partyId"`
	Force   bool   `json:"force"`
}

type changeLeaderBody struct {
	PartyId  uint32 `json:"partyId"`
	LeaderId uint32 `json:"leaderId"`
}

type statusEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	WorldId byte   `json:"worldId"`
	PartyId uint32 `json:"partyId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createdEventBody struct {
}

type joinedEventBody struct {
}

type leftEventBody struct {
}

type expelEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type disbandEventBody struct {
	Members []uint32 `json:"members"`
}

type changeLeaderEventBody struct {
	CharacterId  uint32 `json:"characterId"`
	Disconnected bool   `json:"disconnected"`
}

type errorEventBody struct {
	CharacterName string `json:"characterName"`
}
