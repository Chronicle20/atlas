package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventMemberStatusTopic topic.Token = "EVENT_TOPIC_PARTY_MEMBER_STATUS"
)

const (
	EventPartyMemberStatusTypeLogin        = "LOGIN"
	EventPartyMemberStatusTypeLogout       = "LOGOUT"
	EventPartyMemberStatusTypeLevelChanged = "LEVEL_CHANGED"
	EventPartyMemberStatusTypeJobChanged   = "JOB_CHANGED"
	EventPartyMemberStatusTypeNameChanged  = "NAME_CHANGED"
)

type memberStatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	PartyId     uint32   `json:"partyId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type memberLoginEventBody struct{}

type memberLogoutEventBody struct{}

type memberLevelChangedEventBody struct {
	OldLevel byte   `json:"oldLevel"`
	NewLevel byte   `json:"newLevel"`
	Name     string `json:"name"`
}

type memberJobChangedEventBody struct {
	OldJobId job.Id `json:"oldJobId"`
	NewJobId job.Id `json:"newJobId"`
	Name     string `json:"name"`
}

// memberNameChangedEventBody carries both names because the consumer's job is
// to redraw a window that still shows the old one — the old name is what makes
// the event diagnosable in a log after the registry has already moved on.
type memberNameChangedEventBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}
