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

// CashId is this half's own asset's cash id -- the identifier that survives
// compartment.Release and equipping (unlike AssetId, the locker asset id,
// which stops resolving once the ring leaves the locker) and is what the
// wire needs (design.md §2 OQ-1: GW_ItemSlotBase::liSN). Persisted on Entity
// at purchase time; enrich (processor.go) falls back to looking up AssetId
// in cashshop/inventory/asset only for rows written before this column
// existed, where the stored value is 0.
func (m Model) CashId() int64 {
	return m.cashId
}

// PartnerCashId is the sibling half's CashId, resolved by PairId. Zero when
// the sibling row is missing or its asset cannot be resolved -- this field
// fails soft, never errors (PRD FR-5's channel-side fallback depends on it).
func (m Model) PartnerCashId() int64 {
	return m.partnerCashId
}

// PartnerName is PartnerCharacterId's resolved character name. Empty when
// the character service is unavailable -- fails soft, never errors.
func (m Model) PartnerName() string {
	return m.partnerName
}
