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
	trademsg "atlas-trades/kafka/message/trade"
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

// StagedItem is one item claimed for trade. Under the escrow-at-staging model
// (design §5A) the asset has genuinely LEFT its owner's compartment for
// atlas-trades' own custody store; settlement and teardown both move it out of
// there, never out of an inventory.
//
// tradeSlot is the 1..9 slot of the client's trade dialog — a wire-local
// coordinate with no shared-constants equivalent, so it stays a byte.
//
// escrowId is the custody row this item lives in, and doubles as the transaction
// id of the staging saga that created it (one saga, one row, always). Every
// later operation names it: release at settlement, return at teardown, restore
// on compensation. inventoryType and sourceSlot survive for PROVENANCE only — a
// return does not replay them, because the original slot may well be occupied by
// the time the trade unwinds.
//
// pending is true between submitting the stage and the escrow row being
// confirmed. A pending item HOLDS its dialog slot — the release step unlocks the
// client before the row exists, so without the hold the client could stage a
// second item into the same slot — but is NOT announced to either dialog, since
// a saga that then failed would have shown both clients an item that was never
// escrowed (design §5A.4).
type StagedItem struct {
	tradeSlot     byte
	assetId       asset.Id
	templateId    item.Id
	quantity      asset.Quantity
	inventoryType inventory.Type
	sourceSlot    slot.Position
	escrowId      uuid.UUID
	pending       bool
}

func (s StagedItem) TradeSlot() byte               { return s.tradeSlot }
func (s StagedItem) AssetId() asset.Id             { return s.assetId }
func (s StagedItem) TemplateId() item.Id           { return s.templateId }
func (s StagedItem) Quantity() asset.Quantity      { return s.quantity }
func (s StagedItem) InventoryType() inventory.Type { return s.inventoryType }
func (s StagedItem) SourceSlot() slot.Position     { return s.sourceSlot }
func (s StagedItem) EscrowId() uuid.UUID           { return s.escrowId }
func (s StagedItem) Pending() bool                 { return s.pending }

// Confirmed returns a copy with the pending flag cleared — what the custody
// consumer stores once the escrow row exists.
func (s StagedItem) Confirmed() StagedItem {
	s.pending = false
	return s
}

// NewStagedItem builds one staged item, PENDING.
//
// Deliberately born pending: no path legitimately creates a staged item whose
// escrow row already exists, and defaulting the other way would let a caller
// announce an item to both dialogs before anything had been escrowed.
func NewStagedItem(tradeSlot byte, assetId asset.Id, templateId item.Id, quantity asset.Quantity, inventoryType inventory.Type, sourceSlot slot.Position, escrowId uuid.UUID) StagedItem {
	return StagedItem{
		tradeSlot:     tradeSlot,
		assetId:       assetId,
		templateId:    templateId,
		quantity:      quantity,
		inventoryType: inventoryType,
		sourceSlot:    sourceSlot,
		escrowId:      escrowId,
		pending:       true,
	}
}

// Participant is one side of the trade. position 0 is the room owner, 1 the
// invited character; position drives which side of the client dialog receives
// which update (FR-1.5).
//
// confirmEntries and attestEntries are the two CRC lists the client sends: the
// first with TRADE_CONFIRM, the second with its automatic TRANSACTION reply to
// mode 17 (design §6.2). They are stored as the wire type rather than a
// re-declared domain pair — a {data, crc} tuple carries no domain behaviour of
// its own, and a parallel type would only be converted back at both boundaries.
// Both are empty on GMS <= v79, where the payload has no CRC list at all
// (design §4.4).
//
// mesoTax and mesoDelivered are RESOLVED at settlement time from the tenant's
// tax table and then frozen onto the participant: the saga payload and the
// ledger row must agree, and re-deriving them at ledger-write time would let a
// tenant configuration change land between the two and record figures the
// orchestrator never moved.
type Participant struct {
	characterId    character.Id
	name           string
	position       byte
	confirmed      bool
	attested       bool
	mesoStaged     uint32
	mesoTax        uint32
	mesoDelivered  uint32
	confirmEntries []trademsg.CrcEntry
	attestEntries  []trademsg.CrcEntry
	items          []StagedItem

	// pendingMesoTxId / pendingMesoAmount hold one in-flight meso stake, the
	// meso twin of StagedItem.pending. mesoStaged is NOT advanced until the
	// award_mesos saga completes, so the counterparty's dialog never renders a
	// stake that the debit then failed to take.
	//
	// The staking client is the exception, and deliberately: mode 16 is an
	// assignment, so its own dialog already shows the new number before the
	// server sees the packet. A failure snaps it back with the authoritative
	// re-echo MESO_REFUSED already carries (design §4.2, §5A.5).
	pendingMesoTxId   uuid.UUID
	pendingMesoAmount uint32
}

func (p Participant) CharacterId() character.Id { return p.characterId }
func (p Participant) Name() string              { return p.name }
func (p Participant) Position() byte            { return p.position }
func (p Participant) Confirmed() bool           { return p.confirmed }
func (p Participant) Attested() bool            { return p.attested }
func (p Participant) MesoStaged() uint32        { return p.mesoStaged }

// MesoTax is what this side's staged meso pays in tax, resolved at settlement
// time. Zero until then.
func (p Participant) MesoTax() uint32 { return p.mesoTax }

// MesoDelivered is what the COUNTERPARTY receives out of this side's staged
// meso, resolved at settlement time. Zero until then.
func (p Participant) MesoDelivered() uint32 { return p.mesoDelivered }

// ConfirmEntries returns a copy of the CRC list this side sent with its
// TRADE_CONFIRM.
func (p Participant) ConfirmEntries() []trademsg.CrcEntry { return copyEntries(p.confirmEntries) }

// AttestEntries returns a copy of the CRC list this side sent with its
// TRANSACTION reply.
func (p Participant) AttestEntries() []trademsg.CrcEntry { return copyEntries(p.attestEntries) }

func copyEntries(in []trademsg.CrcEntry) []trademsg.CrcEntry {
	if in == nil {
		return nil
	}
	out := make([]trademsg.CrcEntry, len(in))
	copy(out, in)
	return out
}

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

// WithConfirmEntries returns a copy carrying the CRC list sent with
// TRADE_CONFIRM. The slice is copied, so the caller's may be reused.
func (p Participant) WithConfirmEntries(entries []trademsg.CrcEntry) Participant {
	c := p
	c.confirmEntries = copyEntries(entries)
	return c
}

// WithAttestEntries returns a copy carrying the CRC list sent with the
// TRANSACTION reply.
func (p Participant) WithAttestEntries(entries []trademsg.CrcEntry) Participant {
	c := p
	c.attestEntries = copyEntries(entries)
	return c
}

// WithSettlementMeso returns a copy carrying the resolved tax split for this
// side's staged meso: tax is destroyed, delivered goes to the counterparty.
func (p Participant) WithSettlementMeso(tax uint32, delivered uint32) Participant {
	c := p
	c.mesoTax = tax
	c.mesoDelivered = delivered
	return c
}

func (p Participant) WithMesoStaged(v uint32) Participant { c := p; c.mesoStaged = v; return c }

// PendingMesoTxId is the transaction of the in-flight meso stake, or uuid.Nil
// when none is outstanding.
func (p Participant) PendingMesoTxId() uuid.UUID { return p.pendingMesoTxId }

// PendingMesoAmount is the ABSOLUTE stake the in-flight saga is moving toward.
func (p Participant) PendingMesoAmount() uint32 { return p.pendingMesoAmount }

// WithPendingMeso arms an in-flight meso stake.
func (p Participant) WithPendingMeso(txId uuid.UUID, amount uint32) Participant {
	c := p
	c.pendingMesoTxId = txId
	c.pendingMesoAmount = amount
	return c
}

// WithSettledMeso resolves an in-flight meso stake, committing it when settled
// is true and abandoning it otherwise. It is a no-op unless txId is the stake
// actually outstanding, so a redelivered terminal status cannot commit an amount
// a newer stake has already superseded.
func (p Participant) WithSettledMeso(txId uuid.UUID, settled bool) (Participant, bool) {
	if p.pendingMesoTxId == uuid.Nil || p.pendingMesoTxId != txId {
		return p, false
	}
	c := p
	if settled {
		c.mesoStaged = p.pendingMesoAmount
	}
	c.pendingMesoTxId = uuid.Nil
	c.pendingMesoAmount = 0
	return c, true
}

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

// WithRelocatedItems returns a copy whose staged items carry the corrected
// source slots given, keyed by reservation id. A staged item whose reservation
// is absent from the map is left untouched.

// WithConfirmedItem clears the pending flag on one staged item, identified by
// its escrow row. Returns the participant unchanged if no such item is staged —
// a redelivered custody ack must not resurrect an item a teardown already
// removed.
func (p Participant) WithConfirmedItem(escrowId uuid.UUID) Participant {
	for i := range p.items {
		if p.items[i].escrowId != escrowId {
			continue
		}
		c := p
		c.items = make([]StagedItem, len(p.items))
		copy(c.items, p.items)
		c.items[i] = c.items[i].Confirmed()
		return c
	}
	return p
}

// WithoutItem drops one staged item, identified by its escrow row. Used when a
// stage is refused: the dialog slot it was holding has to come free, or the
// player can never use that slot again.
func (p Participant) WithoutItem(escrowId uuid.UUID) Participant {
	for i := range p.items {
		if p.items[i].escrowId != escrowId {
			continue
		}
		c := p
		c.items = make([]StagedItem, 0, len(p.items)-1)
		c.items = append(c.items, p.items[:i]...)
		c.items = append(c.items, p.items[i+1:]...)
		return c
	}
	return p
}

// ItemByEscrow finds one staged item by its escrow row.
func (p Participant) ItemByEscrow(escrowId uuid.UUID) (StagedItem, bool) {
	for _, i := range p.items {
		if i.escrowId == escrowId {
			return i, true
		}
	}
	return StagedItem{}, false
}

// StagedQuantityFrom totals what this participant has already claimed out of one
// inventory slot. Staging a partial stack twice from the same slot is legal, and
// each claim files its own reservation, so the availability check has to net off
// what THIS room already reserved before comparing against the asset's quantity.
func (p Participant) StagedQuantityFrom(inventoryType inventory.Type, sourceSlot slot.Position) asset.Quantity {
	var total asset.Quantity
	for _, i := range p.items {
		if i.inventoryType == inventoryType && i.sourceSlot == sourceSlot {
			total += i.quantity
		}
	}
	return total
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
// settlementId is the transaction id of the settlement saga this room
// submitted, minted at the SETTLING transition. It is also the ledger's
// idempotency key (FR-5.7) and the only handle the EVENT_TOPIC_SAGA_STATUS
// consumer has for finding the room a terminal saga status belongs to — the
// FAILED body's characterId names the failed expanded step's character, which
// is not a role and may be either participant.
// invitedId is the character the owner's outstanding INVITE named, recorded at
// the OPEN_SOLO -> PENDING_INVITE transition. It is the ADMISSION TICKET: the
// wire handle is the owner's character id and therefore public, so without a
// recorded target any character in the same map could seat themselves in a
// stranger's pending trade by naming the handle. Zero while the room has no
// outstanding invite.
type Room struct {
	id           uuid.UUID
	handle       uint32
	roomType     byte
	f            field.Model
	state        State
	settlementId uuid.UUID
	invitedId    character.Id
	participants []Participant
	createdAt    time.Time
}

func (r Room) Id() uuid.UUID           { return r.id }
func (r Room) Handle() uint32          { return r.handle }
func (r Room) RoomType() byte          { return r.roomType }
func (r Room) Field() field.Model      { return r.f }
func (r Room) State() State            { return r.state }
func (r Room) SettlementId() uuid.UUID { return r.settlementId }
func (r Room) CreatedAt() time.Time    { return r.createdAt }

// InvitedId returns the character the outstanding invite named, or 0 when the
// room has none. Only that character may be seated by ENTER_ROOM.
func (r Room) InvitedId() character.Id { return r.invitedId }

// Admits reports whether characterId may be seated as this room's visitor. It
// is deliberately the ONLY predicate that answers that question, so the invite
// ticket cannot be bypassed by a caller that re-derives seating rules.
func (r Room) Admits(characterId character.Id) bool {
	return r.invitedId != 0 && r.invitedId == characterId
}

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

// WithInvited returns a copy of r whose outstanding invite names characterId.
// A re-issued invite after a decline overwrites the previous target, so the
// declined character loses their ticket.
func (r Room) WithInvited(characterId character.Id) Room {
	c := r
	c.invitedId = characterId
	return c
}

// WithSettlementId returns a copy of r carrying the settlement saga's
// transaction id.
func (r Room) WithSettlementId(id uuid.UUID) Room {
	c := r
	c.settlementId = id
	return c
}

// BothConfirmed reports whether a SEATED PAIR has both confirmed. A solo room
// can never satisfy it: the participant count is part of the test, so a single
// confirmed owner does not read as a confirmed trade.
func (r Room) BothConfirmed() bool {
	if len(r.participants) != 2 {
		return false
	}
	for _, p := range r.participants {
		if !p.confirmed {
			return false
		}
	}
	return true
}

// BothAttested reports whether a seated pair has both replied to the mode-17
// attestation prompt.
func (r Room) BothAttested() bool {
	if len(r.participants) != 2 {
		return false
	}
	for _, p := range r.participants {
		if !p.attested {
			return false
		}
	}
	return true
}

// OrderedParticipants returns the participants by POSITION — owner (0) first,
// visitor (1) second. Every downstream consumer that indexes a pair (the saga
// payload's [2]TradeSettlementSide, the ledger's two sides) needs a stable
// order, and position is the only one the room defines. It is an ORDER, not a
// role assignment: neither index is "the giver", because each side gives its
// own contribution and receives the other's.
func (r Room) OrderedParticipants() []Participant {
	out := r.Participants()
	if len(out) == 2 && out[0].position > out[1].position {
		out[0], out[1] = out[1], out[0]
	}
	return out
}

// WithVisitor returns a copy of r with characterId seated at position 1
// (FR-1.5). Callers must have already rejected a room that is not solo — this
// transform appends unconditionally, so seating a visitor twice would produce
// two position-1 participants. It allocates a fresh participant slice, so it is
// safe to call on a Room obtained from the registry.
//
// Seating SPENDS the invite ticket: invitedId is cleared, so a room that has
// its visitor no longer names an outstanding invite.
func (r Room) WithVisitor(characterId character.Id, name string) Room {
	c := r
	c.invitedId = 0
	c.participants = make([]Participant, len(r.participants), len(r.participants)+1)
	copy(c.participants, r.participants)
	c.participants = append(c.participants, Participant{
		characterId: characterId, name: name, position: 1, items: []StagedItem{},
	})
	return c
}

// WithEachParticipant returns a copy of r with fn applied to EVERY participant.
// It allocates a fresh participant slice, so it is safe to call on a Room
// obtained from the registry.
func (r Room) WithEachParticipant(fn func(Participant) Participant) Room {
	c := r
	c.participants = make([]Participant, len(r.participants))
	copy(c.participants, r.participants)
	for i := range c.participants {
		c.participants[i] = fn(c.participants[i])
	}
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
