package party

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate       = "CREATE"
	CommandPartyJoin         = "JOIN"
	CommandPartyLeave        = "LEAVE"
	CommandPartyChangeLeader = "CHANGE_LEADER"
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
