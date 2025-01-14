package character

const (
	EnvEventTopicCharacterStatus       = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin      = "LOGIN"
	EventCharacterStatusTypeLogout     = "LOGOUT"
	EventCharacterStatusTypeMapChanged = "MAP_CHANGED"

	EnvEventMemberStatusTopic        = "EVENT_TOPIC_PARTY_MEMBER_STATUS"
	EventPartyMemberStatusTypeLogin  = "LOGIN"
	EventPartyMemberStatusTypeLogout = "LOGOUT"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type statusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type memberStatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	PartyId     uint32 `json:"partyId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type memberLoginEventBody struct {
}

type memberLogoutEventBody struct {
}
