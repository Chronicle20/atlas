package invite

const (
	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeAccepted = "ACCEPTED"

	InviteTypeMessenger = "MESSENGER"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	InviteType  string `json:"inviteType"`
	ReferenceId uint32 `json:"referenceId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type acceptedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
