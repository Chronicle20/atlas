package member

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventStatusTopic             = "EVENT_TOPIC_PARTY_MEMBER_STATUS"
	EventPartyStatusTypeLogin       = "LOGIN"
	EventPartyStatusTypeLogout      = "LOGOUT"
	EventPartyStatusTypeNameChanged = "NAME_CHANGED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	PartyId     uint32   `json:"partyId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type LoginEventBody struct{}

type LogoutEventBody struct{}

// NameChangedEventBody mirrors atlas-parties' memberNameChangedEventBody.
type NameChangedEventBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}
