package invite

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeCreate = "CREATE"

	InviteTypeGuild = "GUILD"
)

type commandEvent[E any] struct {
	WorldId    byte   `json:"worldId"`
	InviteType string `json:"inviteType"`
	Type       string `json:"type"`
	Body       E      `json:"body"`
}

type createCommandBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
	ReferenceId  uint32 `json:"referenceId"`
}
