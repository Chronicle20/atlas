package invite

const (
	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeCreated  = "CREATED"
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

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	InviteType  string `json:"inviteType"`
	ReferenceId uint32 `json:"referenceId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type createdEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}

type acceptedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}

type rejectedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
