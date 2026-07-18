package party

const (
	EnvCommandTopic           = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate        = "CREATE"
	CommandPartyJoin          = "JOIN"
	CommandPartyLeave         = "LEAVE"
	CommandPartyChangeLeader  = "CHANGE_LEADER"
	CommandPartyRequestInvite = "REQUEST_INVITE"
)

type commandEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createCommandBody struct{}

type joinCommandBody struct {
	PartyId uint32 `json:"partyId"`
}

type leaveCommandBody struct {
	PartyId uint32 `json:"partyId"`
	Force   bool   `json:"force"`
	// CharacterId identifies the member to remove. For a normal leave it equals
	// the actor; for a forced leave (expel) it is the target being expelled.
	CharacterId uint32 `json:"characterId"`
}

type changeLeaderBody struct {
	PartyId  uint32 `json:"partyId"`
	LeaderId uint32 `json:"leaderId"`
}

type requestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}
