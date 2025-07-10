package member

import (
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_PARTY_MEMBER_STATUS"
	EventPartyStatusTypeLogin  = "LOGIN"
	EventPartyStatusTypeLogout = "LOGOUT"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	PartyId     uint32   `json:"partyId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type LoginEventBody struct {
}

type LogoutEventBody struct {
}
