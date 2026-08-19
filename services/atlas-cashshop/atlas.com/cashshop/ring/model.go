package ring

import (
	"time"

	"github.com/google/uuid"
)

// Type distinguishes a couple ring pair from a friendship ring pair. There is
// no ring-pair type in libs/atlas-constants -- ClassificationRing there is an
// item classification, not a pairing type -- so this is service-local.
type Type string

const (
	TypeCouple     = Type("COUPLE")
	TypeFriendship = Type("FRIENDSHIP")
)

// State records what happened to a pair half without deleting its history
// (FR-RING-9).
type State string

const (
	StateActive  = State("ACTIVE")
	StateBroken  = State("BROKEN")
	StateExpired = State("EXPIRED")
)

// Model is one half of a ring pair.
type Model struct {
	id                 uuid.UUID
	pairId             uuid.UUID
	characterId        uint32
	partnerCharacterId uint32
	assetId            uint32
	itemTemplateId     uint32
	ringType           Type
	state              State
	createdAt          time.Time
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) PairId() uuid.UUID {
	return m.pairId
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) PartnerCharacterId() uint32 {
	return m.partnerCharacterId
}

func (m Model) AssetId() uint32 {
	return m.assetId
}

func (m Model) ItemTemplateId() uint32 {
	return m.itemTemplateId
}

func (m Model) Type() Type {
	return m.ringType
}

func (m Model) State() State {
	return m.state
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
