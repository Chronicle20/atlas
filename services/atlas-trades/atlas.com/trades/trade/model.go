// Package trade holds the trade room domain model, its builder and the
// tenant-partitioned in-memory registry. Room is an immutable snapshot of one
// live trade; the registry (registry.go) owns the only mutable state — every
// mutation swaps an old Room for a new one under a single write lock.
//
// IN-PACKAGE INVARIANT: Registry.Get/GetByMember/GetByHandle/All return Room
// values whose `participants` backing array is SHARED with the registry's
// stored copy. Out-of-package callers cannot reach it (Participants() and
// Items() copy), and every With* transform allocates a fresh slice, so the
// sharing is safe as written. But code inside this package must never write
// through `r.participants[i]` on a Room it got from the registry — that would
// mutate registry state outside the lock. Mutate only via WithParticipant,
// and only inside Registry.Update's callback.
package trade

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// State is the trade room lifecycle (design §3.1).
//
//	CREATE            -> OpenSolo
//	INVITE            -> PendingInvite
//	invite accepted   -> Open                  (staging happens here)
//	both confirmed    -> AwaitingAttestation   (mode 17 broadcast, 5s deadline)
//	both attested     -> Settling              (saga in flight; cancels lose)
type State string

const (
	StateOpenSolo            State = "OPEN_SOLO"
	StatePendingInvite       State = "PENDING_INVITE"
	StateOpen                State = "OPEN"
	StateAwaitingAttestation State = "AWAITING_ATTESTATION"
	StateSettling            State = "SETTLING"
)

// StagedItem is one item claimed for trade. Under the reserve-at-staging model
// (design §5.3) the asset is STILL IN the owner's inventory, held by an
// atlas-inventory reservation; only settlement moves it.
//
// tradeSlot is the 1..9 slot of the client's trade dialog — a wire-local
// coordinate with no shared-constants equivalent, so it stays a byte.
type StagedItem struct {
	tradeSlot     byte
	assetId       asset.Id
	templateId    item.Id
	quantity      asset.Quantity
	inventoryType inventory.Type
	sourceSlot    slot.Position
}

func (s StagedItem) TradeSlot() byte               { return s.tradeSlot }
func (s StagedItem) AssetId() asset.Id             { return s.assetId }
func (s StagedItem) TemplateId() item.Id           { return s.templateId }
func (s StagedItem) Quantity() asset.Quantity      { return s.quantity }
func (s StagedItem) InventoryType() inventory.Type { return s.inventoryType }
func (s StagedItem) SourceSlot() slot.Position     { return s.sourceSlot }

// NewStagedItem builds one staged item. StagedItem is a value type with no
// mutable state, so it needs no builder.
func NewStagedItem(tradeSlot byte, assetId asset.Id, templateId item.Id, quantity asset.Quantity, inventoryType inventory.Type, sourceSlot slot.Position) StagedItem {
	return StagedItem{
		tradeSlot:     tradeSlot,
		assetId:       assetId,
		templateId:    templateId,
		quantity:      quantity,
		inventoryType: inventoryType,
		sourceSlot:    sourceSlot,
	}
}

// Participant is one side of the trade. position 0 is the room owner, 1 the
// invited character; position drives which side of the client dialog receives
// which update (FR-1.5).
type Participant struct {
	characterId character.Id
	name        string
	position    byte
	confirmed   bool
	attested    bool
	mesoStaged  uint32
	items       []StagedItem
}

func (p Participant) CharacterId() character.Id { return p.characterId }
func (p Participant) Name() string              { return p.name }
func (p Participant) Position() byte            { return p.position }
func (p Participant) Confirmed() bool           { return p.confirmed }
func (p Participant) Attested() bool            { return p.attested }
func (p Participant) MesoStaged() uint32        { return p.mesoStaged }

// Items returns a copy of the staged items, so a caller cannot write through
// the returned slice into the participant's state.
func (p Participant) Items() []StagedItem {
	if p.items == nil {
		return nil
	}
	out := make([]StagedItem, len(p.items))
	copy(out, p.items)
	return out
}

func (p Participant) WithConfirmed(v bool) Participant { c := p; c.confirmed = v; return c }

func (p Participant) WithAttested(v bool) Participant { c := p; c.attested = v; return c }

func (p Participant) WithMesoStaged(v uint32) Participant { c := p; c.mesoStaged = v; return c }

// WithItem appends a staged item. Callers must have already rejected a
// duplicate trade slot and enforced the maxStagedItems cap.
func (p Participant) WithItem(i StagedItem) Participant {
	c := p
	c.items = make([]StagedItem, len(p.items), len(p.items)+1)
	copy(c.items, p.items)
	c.items = append(c.items, i)
	return c
}

// HasTradeSlot reports whether the given 1..9 trade slot is already occupied
// (FR-3.3).
func (p Participant) HasTradeSlot(tradeSlot byte) bool {
	for _, i := range p.items {
		if i.tradeSlot == tradeSlot {
			return true
		}
	}
	return false
}

// Room is a live trade room. It carries two ids (design §2.3): Id is the REST
// identity and registry key, Handle is the uint32 wire serial the client's
// invite carries (invite.CreateCommandBody.ReferenceId is invite.Id = uint32,
// so a uuid does not fit). Handle is set to the owner's character id, matching
// the existing mini-room convention in atlas-channel — but it is a wire serial,
// not a character reference, so it is a plain uint32 rather than character.Id:
// a later task may hand SetHandle a value that is not anyone's character id.
//
// roomType is a miniroom type byte — miniroom.Trade (3) or miniroom.CashTrade
// (6) from libs/atlas-constants/miniroom.
type Room struct {
	id           uuid.UUID
	handle       uint32
	roomType     byte
	f            field.Model
	state        State
	participants []Participant
	createdAt    time.Time
}

func (r Room) Id() uuid.UUID        { return r.id }
func (r Room) Handle() uint32       { return r.handle }
func (r Room) RoomType() byte       { return r.roomType }
func (r Room) Field() field.Model   { return r.f }
func (r Room) State() State         { return r.state }
func (r Room) CreatedAt() time.Time { return r.createdAt }

// Participants returns a copy of the participant list, so a caller cannot
// write through the returned slice into the room's state.
func (r Room) Participants() []Participant {
	if r.participants == nil {
		return nil
	}
	out := make([]Participant, len(r.participants))
	copy(out, r.participants)
	return out
}

// OwnerId returns position 0's character id.
func (r Room) OwnerId() character.Id {
	for _, p := range r.participants {
		if p.position == 0 {
			return p.characterId
		}
	}
	return 0
}

// VisitorId returns position 1's character id, or 0 when the room is solo.
func (r Room) VisitorId() character.Id {
	for _, p := range r.participants {
		if p.position == 1 {
			return p.characterId
		}
	}
	return 0
}

// ParticipantFor returns the participant acting as characterId.
func (r Room) ParticipantFor(characterId character.Id) (Participant, bool) {
	for _, p := range r.participants {
		if p.characterId == characterId {
			return p, true
		}
	}
	return Participant{}, false
}

// Frozen reports whether staging is closed. From the moment the FIRST side
// confirms, the room rejects PUT_ITEM, ADD_MESO and any further CONFIRM from
// either side (FR-3.6, design §3.2).
func (r Room) Frozen() bool {
	if r.state != StateOpen {
		return true
	}
	for _, p := range r.participants {
		if p.confirmed {
			return true
		}
	}
	return false
}

// WithState returns a copy of r in the given state.
func (r Room) WithState(s State) Room {
	c := r
	c.state = s
	return c
}

// WithParticipant returns a copy of r with the participant at `position`
// replaced by fn's result. Participants are value types, so fn receives and
// returns a copy — this is the only way room state mutates.
func (r Room) WithParticipant(position byte, fn func(Participant) Participant) Room {
	c := r
	c.participants = make([]Participant, len(r.participants))
	copy(c.participants, r.participants)
	for i := range c.participants {
		if c.participants[i].position == position {
			c.participants[i] = fn(c.participants[i])
		}
	}
	return c
}
