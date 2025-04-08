package invite

const (
	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeAccepted = "ACCEPTED"
	EventInviteStatusTypeRejected = "REJECTED"

	InviteTypeBuddy = "BUDDY"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	InviteType  string `json:"inviteType"`
	ReferenceId uint32 `json:"referenceId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type AcceptedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}

type RejectedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
