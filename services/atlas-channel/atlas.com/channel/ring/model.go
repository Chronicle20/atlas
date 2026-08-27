package ring

import "github.com/google/uuid"

// Type distinguishes a couple ring pair from a friendship ring pair. There is
// no ring-pair type in libs/atlas-constants -- ClassificationRing there
// (libs/atlas-constants/item/constants.go:24) is an item classification, not
// a pairing type -- so this is service-local, re-declared to match
// atlas-cashshop/ring/model.go:14-17.
type Type string

const (
	TypeCouple     = Type("COUPLE")
	TypeFriendship = Type("FRIENDSHIP")
)

// State records what happened to a pair half without deleting its history,
// re-declared to match atlas-cashshop/ring/model.go:23-26.
type State string

const (
	StateActive  = State("ACTIVE")
	StateBroken  = State("BROKEN")
	StateExpired = State("EXPIRED")
)

// Model is the channel-side read-only view of one half of a ring pair,
// consumed from atlas-cashshop's GET /rings route (task-269 task 8). There is
// no write path here: a ring pair is created only by the purchase
// transaction on atlas-cashshop.
type Model struct {
	id                 uuid.UUID
	pairId             uuid.UUID
	characterId        uint32
	partnerCharacterId uint32
	itemTemplateId     uint32
	ringType           Type
	state              State
	cashId             int64
	partnerCashId      int64
	partnerName        string
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

func (m Model) ItemTemplateId() uint32 {
	return m.itemTemplateId
}

func (m Model) Type() Type {
	return m.ringType
}

func (m Model) State() State {
	return m.state
}

// CashId is this half's own locker asset's cash id (design.md OQ-1), the
// identifier the wire needs. Resolved server-side at read time; see
// atlas-cashshop/ring/model.go:81-88.
func (m Model) CashId() int64 {
	return m.cashId
}

// PartnerCashId is the sibling half's CashId, zero when the sibling row is
// missing or its asset cannot be resolved -- server-side fail-soft, never an
// error; see atlas-cashshop/ring/model.go:90-95.
func (m Model) PartnerCashId() int64 {
	return m.partnerCashId
}

// PartnerName is PartnerCharacterId's resolved character name, empty when
// the character service was unavailable server-side -- fail-soft, never an
// error; see atlas-cashshop/ring/model.go:97-101.
func (m Model) PartnerName() string {
	return m.partnerName
}
