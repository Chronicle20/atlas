package invite

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeAccept = "ACCEPT"
	CommandInviteTypeReject = "REJECT"

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
	WorldId    byte   `json:"worldId"`
	InviteType string `json:"inviteType"`
	Type       string `json:"type"`
	Body       E      `json:"body"`
}

type acceptCommandBody struct {
	TargetId    uint32 `json:"targetId"`
	ReferenceId uint32 `json:"referenceId"`
}

type rejectCommandBody struct {
	TargetId     uint32 `json:"targetId"`
	OriginatorId uint32 `json:"originatorId"`
}
