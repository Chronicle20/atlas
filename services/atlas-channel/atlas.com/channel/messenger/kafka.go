package messenger

const (
	EnvCommandTopic               = "COMMAND_TOPIC_MESSENGER"
	CommandMessengerCreate        = "CREATE"
	CommandMessengerLeave         = "LEAVE"
	CommandMessengerRequestInvite = "REQUEST_INVITE"
)

type commandEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createCommandBody struct {
}

type leaveCommandBody struct {
	MessengerId uint32 `json:"messengerId"`
}

type requestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}
