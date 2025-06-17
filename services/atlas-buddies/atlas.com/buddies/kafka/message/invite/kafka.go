package invite

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeCreate = "CREATE"
	CommandInviteTypeReject = "REJECT"

	InviteTypeBuddy = "BUDDY"
)

type Command[E any] struct {
	WorldId    byte   `json:"worldId"`
	InviteType string `json:"inviteType"`
	Type       string `json:"type"`
	Body       E      `json:"body"`
}

type CreateCommandBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
	ReferenceId  uint32 `json:"referenceId"`
}

type RejectCommandBody struct {
	TargetId     uint32 `json:"targetId"`
	OriginatorId uint32 `json:"originatorId"`
}

const (
	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeAccepted = "ACCEPTED"
	EventInviteStatusTypeRejected = "REJECTED"
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
