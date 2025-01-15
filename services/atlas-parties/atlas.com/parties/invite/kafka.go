package invite

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeCreate = "CREATE"

	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeAccepted = "ACCEPTED"
	EventInviteStatusTypeRejected = "REJECTED"

	InviteTypeBuddy        = "BUDDY"
	InviteTypeFamily       = "FAMILY"
	InviteTypeFamilySummon = "FAMILY_SUMMON"
	InviteTypeMessenger    = "MESSENGER"
	InviteTypeTrade        = "TRADE"
	InviteTypeParty        = "PARTY"
	InviteTypeGuild        = "GUILD"
	InviteTypeAlliance     = "ALLIANCE"
)

type commandEvent[E any] struct {
	WorldId byte   `json:"worldId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createCommandBody struct {
	InviteType   string `json:"inviteType"`
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
	ReferenceId  uint32 `json:"referenceId"`
}

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

type rejectedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
