package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/kafka/message"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// sagaFailedReason labels the settlement-failure metric when the orchestrator,
// not a pre-check, refused the trade. Pre-check refusals carry their own
// leaveReason key.
const sagaFailedReason = "SAGA_FAILED"

// --- the attestation deadline -------------------------------------------------

// attestationTimers holds one armed deadline per room in AWAITING_ATTESTATION
// (design §3.1: the attestation is defence in depth, not a liveness
// dependency — a client that never replies must not wedge a trade).
//
// Cancelling a timer is an OPTIMISATION, not a correctness requirement:
// ExpireAttestation re-reads the room and does nothing unless it is still
// awaiting attestation, so a stray wakeup after the room settled or was torn
// down is already a no-op. The registry exists so the common case does not leak
// a goroutine per trade for the length of the deadline.
type attestationTimers struct {
	mutex sync.Mutex
	stops map[tenant.Model]map[uuid.UUID]chan struct{}
}

func newAttestationTimers() *attestationTimers {
	return &attestationTimers{stops: make(map[tenant.Model]map[uuid.UUID]chan struct{})}
}

var (
	attestationTimerRegistry *attestationTimers
	attestationTimerOnce     sync.Once
)

// GetAttestationTimers returns the process-wide deadline registry.
func GetAttestationTimers() *attestationTimers {
	attestationTimerOnce.Do(func() { attestationTimerRegistry = newAttestationTimers() })
	return attestationTimerRegistry
}

// Arm schedules fire for one room, replacing any deadline already armed for it.
//
// ctx is deliberately NOT the command's context. A Kafka handler's context is
// done the moment the handler returns, so arming against it would fire the
// deadline immediately-and-never: the goroutine would exit before the timer.
func (r *attestationTimers) Arm(l logrus.FieldLogger, ctx context.Context, t tenant.Model, roomId uuid.UUID, d time.Duration, fire func()) {
	stop := make(chan struct{})

	r.mutex.Lock()
	if r.stops[t] == nil {
		r.stops[t] = make(map[uuid.UUID]chan struct{})
	}
	if existing, ok := r.stops[t][roomId]; ok {
		close(existing)
	}
	r.stops[t][roomId] = stop
	r.mutex.Unlock()

	routine.Go(l, ctx, func(c context.Context) {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
			r.forget(t, roomId, stop)
			fire()
		case <-stop:
		case <-c.Done():
			r.forget(t, roomId, stop)
		}
	})
}

// Cancel disarms the room's deadline, if one is armed.
func (r *attestationTimers) Cancel(t tenant.Model, roomId uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if stop, ok := r.stops[t][roomId]; ok {
		close(stop)
		delete(r.stops[t], roomId)
	}
}

// isArmed reports whether a deadline is currently armed for the room. Nothing
// in the service branches on it — the settlement flow is driven by room state,
// never by timer bookkeeping — but a deadline that silently failed to arm
// wedges the room permanently and is otherwise only observable by waiting it
// out, so it is exposed for the test that pins the arming.
func (r *attestationTimers) isArmed(t tenant.Model, roomId uuid.UUID) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.stops[t][roomId]
	return ok
}

// StopAll disarms every deadline. main registers it as a teardown so shutdown
// does not wait on a sleeping timer.
func (r *attestationTimers) StopAll() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for t, rooms := range r.stops {
		for id, stop := range rooms {
			close(stop)
			delete(rooms, id)
		}
		delete(r.stops, t)
	}
}

// forget drops the registry entry for a deadline that has fired, but only if it
// is still THIS deadline's — a re-arm between the fire and the lock installed a
// newer channel that must survive.
func (r *attestationTimers) forget(t tenant.Model, roomId uuid.UUID, stop chan struct{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if held, ok := r.stops[t][roomId]; ok && held == stop {
		delete(r.stops[t], roomId)
	}
}

// detached returns a copy of the processor whose context carries the tenant but
// no cancellation and no deadline of its own. The attestation timer fires long
// after the command that armed it returned, so the fired work cannot run on
// that command's context — and it must not run on its transaction either, which
// is why Confirm arms from OUTSIDE emit, where p.db is still the root handle.
func (p *ProcessorImpl) detached() *ProcessorImpl {
	c := *p
	c.ctx = tenant.WithContext(context.Background(), p.t)
	return &c
}

// armAttestationDeadline schedules ExpireAttestation for the room.
func (p *ProcessorImpl) armAttestationDeadline(roomId uuid.UUID) {
	d := p.cfg.Get(p.l, p.ctx).AttestationTimeout()
	dp := p.detached()
	p.timers.Arm(p.l, dp.ctx, p.t, roomId, d, func() {
		if err := dp.ExpireAttestation(uuid.New(), roomId); err != nil {
			dp.l.WithError(err).Errorf("Unable to expire the attestation deadline of trade room [%s].", roomId.String())
		}
	})
}

// --- CONFIRM ------------------------------------------------------------------

// Confirm records one side pressing Trade (FR-5.1). See the Processor interface
// for the contract; the ordering below is design §6.2's.
func (p *ProcessorImpl) Confirm(txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) error {
	var arm uuid.UUID
	err := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		var cerr error
		arm, cerr = txp.confirm(mb, txId, characterId, entries)
		return cerr
	})
	// Armed OUTSIDE emit, on the root processor: the deadline outlives both the
	// transaction and the command's context.
	//
	// Armed REGARDLESS OF err, because the registry swap to AWAITING_ATTESTATION
	// is in-memory and a rolled-back transaction does not undo it (see emit).
	// A room left in that state with no deadline is wedged for good: no mode 17
	// ever reaches the clients, so no attestation can arrive, nothing else
	// settles it, and RefreshReservations keeps both sides' holds alive
	// indefinitely. confirm therefore reports the room to arm as soon as the
	// swap lands, before it buffers anything.
	if arm != uuid.Nil {
		p.armAttestationDeadline(arm)
	}
	return err
}

// confirm returns the id of the room whose attestation deadline must be armed,
// or uuid.Nil when this confirm was the first of the two. The id is set the
// moment the transition lands and is returned ALONGSIDE any later error, so a
// failed emit cannot leave an unarmed AWAITING_ATTESTATION room behind.
func (p *ProcessorImpl) confirm(mb *message.Buffer, txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) (uuid.UUID, error) {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued CONFIRM without a trade room. Dropping.", characterId)
		return uuid.Nil, nil
	}
	// A confirm is only legal from OPEN. Past that the room is either already
	// awaiting attestation or settling, and the reference client blocks a second
	// press locally (design §3.2), so reaching here means a modified client.
	if room.State() != StateOpen {
		p.l.Warnf("Character [%d] issued CONFIRM against trade room [%s] in state [%s]. Dropping.", characterId, room.Id().String(), room.State())
		return uuid.Nil, nil
	}
	pt, ok := room.ParticipantFor(characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued CONFIRM but is not seated in room [%s]. Dropping.", characterId, room.Id().String())
		return uuid.Nil, nil
	}
	if pt.Confirmed() {
		p.l.Warnf("Character [%d] issued a second CONFIRM against trade room [%s]. Dropping: a repeated confirm cannot stand in for the counterparty's.", characterId, room.Id().String())
		return uuid.Nil, nil
	}

	// One compare-and-set does both halves: the participant's confirm and, when
	// it completes the pair, the transition. Splitting them would let two
	// simultaneous confirms each observe the other as unconfirmed.
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if cur.State() != StateOpen {
			return Room{}, ErrRoomFrozen
		}
		cp, found := cur.ParticipantFor(characterId)
		if !found {
			return Room{}, ErrRoomNotFound
		}
		if cp.Confirmed() {
			return Room{}, ErrRoomFrozen
		}
		next := cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithConfirmed(true).WithConfirmEntries(entries)
		})
		if next.BothConfirmed() {
			next = next.WithState(StateAwaitingAttestation)
		}
		return next, nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Character [%d]'s CONFIRM lost a race. Dropping.", characterId)
		return uuid.Nil, nil
	}

	// Decided before ANY buffering: from here on every return carries it.
	var arm uuid.UUID
	if updated.State() == StateAwaitingAttestation {
		arm = updated.Id()
	}

	if err = mb.Put(trademsg.EnvEventTopicStatus, participantConfirmedProvider(txId, updated, characterId, pt.Position())); err != nil {
		return arm, err
	}
	if arm == uuid.Nil {
		return uuid.Nil, nil
	}

	p.l.WithFields(p.roomFields(updated)).Infof("Both sides of trade room [%s] confirmed; requesting attestation.", updated.Id().String())
	if err = mb.Put(trademsg.EnvEventTopicStatus, attestationRequestedProvider(txId, updated, characterId)); err != nil {
		return arm, err
	}
	return arm, nil
}

// --- TRANSACTION (the attestation reply) --------------------------------------

// Attest records one side's CRC attestation and, once both have replied,
// settles.
func (p *ProcessorImpl) Attest(txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) error {
	return p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.attest(mb, txId, characterId, entries)
	})
}

func (p *ProcessorImpl) attest(mb *message.Buffer, txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued TRANSACTION without a trade room. Dropping.", characterId)
		return nil
	}
	if room.State() != StateAwaitingAttestation {
		p.l.Debugf("Character [%d] issued TRANSACTION against trade room [%s] in state [%s]. Dropping.", characterId, room.Id().String(), room.State())
		return nil
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if cur.State() != StateAwaitingAttestation {
			return Room{}, ErrRoomFrozen
		}
		cp, found := cur.ParticipantFor(characterId)
		if !found {
			return Room{}, ErrRoomNotFound
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithAttested(true).WithAttestEntries(entries)
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Character [%d]'s TRANSACTION lost a race. Dropping.", characterId)
		return nil
	}
	if !updated.BothAttested() {
		return nil
	}
	return p.settle(mb, txId, updated)
}

// ExpireAttestation settles a room whose attestation deadline lapsed, using
// whatever attestation arrived (design §3.1). The side that never replied is
// simply not CRC-checked: its TRADE_CONFIRM list is the only evidence there is,
// and comparing that list to itself would be a tautology.
func (p *ProcessorImpl) ExpireAttestation(txId uuid.UUID, roomId uuid.UUID) error {
	return p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.expireAttestation(mb, txId, roomId)
	})
}

func (p *ProcessorImpl) expireAttestation(mb *message.Buffer, txId uuid.UUID, roomId uuid.UUID) error {
	room, ok := p.reg.Get(p.t, roomId)
	if !ok {
		p.l.Debugf("Attestation deadline for trade room [%s] fired after the room was gone. Ignoring.", roomId.String())
		return nil
	}
	if room.State() != StateAwaitingAttestation {
		p.l.Debugf("Attestation deadline for trade room [%s] fired in state [%s]. Ignoring.", roomId.String(), room.State())
		return nil
	}
	p.l.WithFields(p.roomFields(room)).Infof("Attestation deadline lapsed for trade room [%s]; settling on the confirm lists.", roomId.String())
	return p.settle(mb, txId, room)
}

// --- settlement ---------------------------------------------------------------

// settle runs the design §6.1 pre-checks against FRESH reads, then submits one
// trade_settlement saga whose transaction id is also the ledger's idempotency
// key (FR-5.7).
//
// A refusal TEARS THE ROOM DOWN (design §6.1's correction of PRD FR-4.9):
// CTradingRoomDlg::OnLeave closes the dialog before showing any of these
// notices, so there is no client state in which the room survives a status
// 8/9/13. Under the reserve-at-staging model nothing needs reverting anyway.
//
// Failure -> LEAVE status mapping:
//
//	(1) free slots       -> TRADE_CANNOT_CARRY  (9)
//	(2) meso cap         -> TRADE_CANNOT_CARRY  (9)
//	(3) reservation lost -> TRADE_FAILED        (8)
//	(4) CRC mismatch     -> TRADE_CRC_FAILED    (13)
func (p *ProcessorImpl) settle(mb *message.Buffer, txId uuid.UUID, room Room) error {
	if len(room.Participants()) != 2 {
		p.l.Errorf("Trade room [%s] reached settlement with [%d] participants. Tearing it down.", room.Id().String(), len(room.Participants()))
		return p.teardownRoom(mb, txId, room, room.OwnerId(), ReasonTradeFailed)
	}
	cfg := p.cfg.Get(p.l, p.ctx)
	cache := p.newCompartmentCache()

	// OBLIGATION: re-resolve every staged slot HERE, not just on the refresh
	// tick. The refresh corrects a relocated item only on a tick boundary, and
	// the CONFIRM freeze stops trade staging, not inventory moves — so an item
	// dragged in the last tick interval before CONFIRM would otherwise be
	// settled from a stale slot. The orchestrator's expander resolves the asset
	// BY SLOT and then rejects a slot holding a different instance
	// (saga/processor.go:1477-1481), so a stale slot is not a silent wrong-asset
	// transfer — it is a saga failure, i.e. LEAVE 8 on a trade that was fine.
	corrections := p.stagedSlotCorrections(cache, room)
	resolved := room
	if len(corrections) > 0 {
		resolved = room.WithEachParticipant(func(v Participant) Participant {
			return v.WithRelocatedItems(corrections)
		})
	}

	ordered := resolved.OrderedParticipants()
	var taxes [2]mesoSplit
	for i, pt := range ordered {
		tax, delivered := configuration.Tax(cfg, pt.MesoStaged())
		taxes[i] = mesoSplit{tax: tax, delivered: delivered}
	}

	if reason, ok := p.preCheck(cache, ordered, taxes); !ok {
		p.l.WithFields(p.roomFields(resolved)).Infof("Refusing to settle trade room [%s]: [%s].", resolved.Id().String(), reason)
		recordSettlementFailed(p.t, reason)
		return p.teardownRoom(mb, txId, resolved, resolved.OwnerId(), reason)
	}

	settlementId := uuid.New()
	// Compare-and-set. Another goroutine that already drove this room to
	// SETTLING wins, and this attempt becomes a no-op rather than a second saga
	// (design §12). The corrections and the resolved tax split are written back
	// in the same step, so the payload built below and the ledger row written at
	// terminal success are derived from one and the same room.
	updated, err := p.reg.Update(p.t, resolved.Id(), func(cur Room) (Room, error) {
		if cur.State() != StateAwaitingAttestation {
			return Room{}, ErrRoomFrozen
		}
		next := cur
		if len(corrections) > 0 {
			next = next.WithEachParticipant(func(v Participant) Participant {
				return v.WithRelocatedItems(corrections)
			})
		}
		for i, pt := range ordered {
			split := taxes[i]
			next = next.WithParticipant(pt.Position(), func(v Participant) Participant {
				return v.WithSettlementMeso(split.tax, split.delivered)
			})
		}
		return next.WithState(StateSettling).WithSettlementId(settlementId), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Trade room [%s] was already driven to settlement by another command. Dropping.", resolved.Id().String())
		return nil
	}

	p.timers.Cancel(p.t, updated.Id())

	// The DURABLE record is written in the SAME transaction that enqueues the
	// saga command, so submission and the record cannot diverge: either both
	// commit or neither does. Without it a restart before the terminal status
	// loses the room, and with it the trade that has already executed — no
	// ledger row, no SETTLED (FR-7.1).
	if _, err = settlement.NewProcessor(p.l, p.ctx, p.db).Submit(settlementRecordFor(settlementId, updated)); err != nil {
		return err
	}

	p.l.WithFields(p.roomFields(updated)).WithField("settlement_id", settlementId.String()).Infof("Submitting settlement for trade room [%s].", updated.Id().String())
	return p.sgp.Settle(mb)(settlementId, settlementPayload(settlementId, updated))
}

// mesoSplit is one side's resolved tax outcome: tax is destroyed, delivered
// goes to the counterparty.
type mesoSplit struct {
	tax       uint32
	delivered uint32
}

// stagedSlotCorrections re-resolves every staged item against its asset id and
// returns the slots that moved, keyed by reservation id. The REST reads happen
// here, never inside a Registry.Update closure, which runs under the write lock
// and must stay pure.
func (p *ProcessorImpl) stagedSlotCorrections(cache *compartmentCache, room Room) map[uuid.UUID]slot.Position {
	corrections := make(map[uuid.UUID]slot.Position)
	for _, pt := range room.Participants() {
		for _, i := range pt.Items() {
			if got := p.resolveStagedSlot(cache, pt.CharacterId(), i); got != i.SourceSlot() {
				corrections[i.ReservationId()] = got
			}
		}
	}
	return corrections
}

// settlementPayload projects the room onto the composite the orchestrator
// expands. Sides are ordered by seat, which the [2]TradeSettlementSide array
// requires; that order carries no role meaning (see Room.OrderedParticipants).
func settlementPayload(settlementId uuid.UUID, room Room) sharedsaga.TradeSettlementPayload {
	payload := sharedsaga.TradeSettlementPayload{
		TransactionId: settlementId,
		WorldId:       room.Field().WorldId(),
		ChannelId:     room.Field().ChannelId(),
		RoomType:      room.RoomType(),
	}
	for i, pt := range room.OrderedParticipants() {
		side := sharedsaga.TradeSettlementSide{
			CharacterId:   pt.CharacterId(),
			MesoStaged:    pt.MesoStaged(),
			MesoTax:       pt.MesoTax(),
			MesoDelivered: pt.MesoDelivered(),
		}
		for _, it := range pt.Items() {
			side.Items = append(side.Items, sharedsaga.TradeSettlementItem{
				InventoryType: it.InventoryType(),
				SourceSlot:    it.SourceSlot(),
				AssetId:       it.AssetId(),
				TemplateId:    it.TemplateId(),
				Quantity:      it.Quantity(),
			})
		}
		payload.Sides[i] = side
	}
	return payload
}

// preCheck runs design §6.1's four checks against fresh reads and returns the
// leaveReason key to refuse with, or ("", true) to proceed. ordered is the
// slot-corrected participant pair; taxes[i] is ordered[i]'s resolved split.
func (p *ProcessorImpl) preCheck(cache *compartmentCache, ordered []Participant, taxes [2]mesoSplit) (string, bool) {
	// (1) Each side has room for what it is about to receive.
	for i, receiver := range ordered {
		if reason, ok := p.canCarry(cache, receiver, ordered[1-i].Items()); !ok {
			return reason, false
		}
	}

	// (2) Each side's meso stays inside [0, cap] after paying out and being
	//     paid. The counterparty's DELIVERED amount is what arrives, not their
	//     staged amount — the tax never reaches anybody.
	for i, side := range ordered {
		if reason, ok := p.mesoSettleable(side, taxes[1-i].delivered); !ok {
			return reason, false
		}
	}

	// (3) Every staged asset is still where the reservation should be holding
	//     it, at the staged quantity.
	for _, side := range ordered {
		if reason, ok := p.stagedAssetsIntact(cache, side); !ok {
			return reason, false
		}
	}

	// (4) The attestation matches the confirm.
	for _, side := range ordered {
		if !entriesMatch(side.ConfirmEntries(), side.AttestEntries()) {
			p.l.Warnf("Character [%d]'s TRANSACTION CRC list does not match the list it sent with TRADE_CONFIRM.", side.CharacterId())
			return ReasonTradeCrcFailed, false
		}
	}

	return "", true
}

// canCarry reports whether the receiver has room for `incoming`, simulating
// atlas-inventory's Accept: it merges into the first existing stack of the same
// template that has room, spilling any remainder into fresh slots
// (services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1666-1730).
//
// The receiver's OWN outgoing items are netted off first, because all releases
// precede all accepts in the expanded saga (design §6.3) — a slot the receiver
// empties is available to what it is about to be handed.
//
// Every read failure is a refusal. An unreadable compartment or an unreadable
// slotMax cannot be defaulted: assuming room is how a trade overflows an
// inventory, and this check exists precisely to stop that reaching the saga.
func (p *ProcessorImpl) canCarry(cache *compartmentCache, receiver Participant, incoming []StagedItem) (string, bool) {
	byType := make(map[inventory.Type][]StagedItem)
	for _, it := range incoming {
		byType[it.InventoryType()] = append(byType[it.InventoryType()], it)
	}

	for it, items := range byType {
		c, err := cache.get(receiver.CharacterId(), it)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to read character [%d]'s compartment [%d] at settlement. Refusing the trade rather than assuming it fits.", receiver.CharacterId(), it)
			return ReasonTradeCannotCarry, false
		}

		free, stacks := p.projectCompartment(c, receiver, it)
		for _, in := range items {
			slotMax, serr := p.idp.SlotMax(it, in.TemplateId())
			if serr != nil {
				p.l.WithError(serr).Errorf("Unable to read the slotMax of item [%d]. Refusing the trade rather than guessing how many slots it needs.", in.TemplateId())
				return ReasonTradeCannotCarry, false
			}
			remaining := uint32(in.Quantity())
			for si := range stacks {
				if remaining == 0 {
					break
				}
				if stacks[si].templateId != uint32(in.TemplateId()) || stacks[si].quantity >= slotMax {
					continue
				}
				take := min(slotMax-stacks[si].quantity, remaining)
				stacks[si].quantity += take
				remaining -= take
			}
			for remaining > 0 {
				if free == 0 {
					p.l.Infof("Character [%d] has no free slot in compartment [%d] for the [%d] of item [%d] they are receiving.", receiver.CharacterId(), it, in.Quantity(), in.TemplateId())
					return ReasonTradeCannotCarry, false
				}
				free--
				take := min(slotMax, remaining)
				stacks = append(stacks, simulatedStack{templateId: uint32(in.TemplateId()), quantity: take})
				remaining -= take
			}
		}
	}
	return "", true
}

// simulatedStack is one occupied slot in the free-slot simulation.
type simulatedStack struct {
	templateId uint32
	quantity   uint32
}

// projectCompartment returns the receiver's free slot count and its occupied
// stacks, with what the receiver is GIVING AWAY out of this compartment already
// netted off. Negative slots are the equipped positions and occupy no bag slot
// (see restriction.go's assetView.SourceSlot).
func (p *ProcessorImpl) projectCompartment(c inventorydata.Model, receiver Participant, it inventory.Type) (uint32, []simulatedStack) {
	occupied := uint32(0)
	stacks := make([]simulatedStack, 0, len(c.Assets()))
	for _, a := range c.Assets() {
		if a.Slot() < 1 {
			continue
		}
		occupied++
		given := receiver.StagedQuantityFrom(it, a.Slot())
		if given >= a.Quantity() {
			// The whole stack leaves; the slot frees.
			occupied--
			continue
		}
		stacks = append(stacks, simulatedStack{templateId: uint32(a.TemplateId()), quantity: uint32(a.Quantity() - given)})
	}
	if c.Capacity() <= occupied {
		return 0, stacks
	}
	return c.Capacity() - occupied, stacks
}

// mesoSettleable reports whether the side can pay out what it staged and take
// in what the counterparty delivers without leaving [0, cap]. The balance is
// read FRESH: a value captured when the meso was staged would let a player
// stage meso and spend it before confirming.
func (p *ProcessorImpl) mesoSettleable(side Participant, incoming uint32) (string, bool) {
	if side.MesoStaged() == 0 && incoming == 0 {
		return "", true
	}
	cm, err := p.cp.GetById(side.CharacterId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read character [%d]'s meso at settlement. Refusing the trade.", side.CharacterId())
		return ReasonTradeFailed, false
	}
	if cm.Meso() < side.MesoStaged() {
		p.l.Infof("Character [%d] staged [%d] meso but now holds [%d]. Refusing the settlement.", side.CharacterId(), side.MesoStaged(), cm.Meso())
		return ReasonTradeCannotCarry, false
	}
	if uint64(cm.Meso())-uint64(side.MesoStaged())+uint64(incoming) > maxStageableMeso {
		p.l.Infof("Character [%d] would hold more than [%d] meso after the trade. Refusing the settlement.", side.CharacterId(), uint32(maxStageableMeso))
		return ReasonTradeCannotCarry, false
	}
	return "", true
}

// stagedAssetsIntact reports whether every asset this side staged is still
// present, still the template it was staged as, and still holds at least what
// was claimed out of it.
//
// This is design §6.1's check 3 as far as the reads available allow. The
// RESERVATION itself is not observable from here — atlas-inventory exposes no
// read of its reservation registry — so what is verified is the state the
// reservation exists to protect. An asset that vanished or shrank is exactly
// the outcome a lapsed or never-filed hold produces, which is why it increments
// the reservation-expired counter.
func (p *ProcessorImpl) stagedAssetsIntact(cache *compartmentCache, side Participant) (string, bool) {
	for _, i := range side.Items() {
		c, err := cache.get(side.CharacterId(), i.InventoryType())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to re-read character [%d]'s compartment [%d] at settlement. Refusing the trade.", side.CharacterId(), i.InventoryType())
			return ReasonTradeFailed, false
		}
		a, ok := c.FindById(i.AssetId())
		if !ok {
			p.l.Warnf("Staged asset [%d] is no longer in character [%d]'s compartment [%d]. Refusing the settlement.", i.AssetId(), side.CharacterId(), i.InventoryType())
			recordReservationExpired(p.t)
			return ReasonTradeFailed, false
		}
		if a.TemplateId() != i.TemplateId() {
			p.l.Warnf("Staged asset [%d] for character [%d] is now template [%d], staged as [%d]. Refusing the settlement.", i.AssetId(), side.CharacterId(), a.TemplateId(), i.TemplateId())
			return ReasonTradeFailed, false
		}
		if claimed := side.StagedQuantityFrom(i.InventoryType(), i.SourceSlot()); a.Quantity() < claimed {
			p.l.Warnf("Character [%d] staged [%d] out of asset [%d], which now holds [%d]. Refusing the settlement.", side.CharacterId(), claimed, i.AssetId(), a.Quantity())
			recordReservationExpired(p.t)
			return ReasonTradeFailed, false
		}
	}
	return "", true
}

// entriesMatch reports whether the attestation reproduces the confirm's CRC
// list.
//
// An EMPTY attestation always matches. That is not laxity: the CRC list is
// absent from the wire entirely on GMS <= v79 (the serverbound tradeCrcPresent
// gate, design §4.4), and the timeout path settles with no attestation at all —
// in both cases there is nothing to compare, and refusing would break every
// legacy version and every slow client. A client that wants to skip the check
// can only do so by sending nothing, which is indistinguishable from not
// replying, which the deadline already settles.
//
// The comparison is order-insensitive. The client builds the list by walking
// its dialog slots, so a reordering is not evidence of tampering; a changed or
// missing pair is.
func entriesMatch(confirmed []trademsg.CrcEntry, attested []trademsg.CrcEntry) bool {
	if len(attested) == 0 {
		return true
	}
	if len(attested) != len(confirmed) {
		return false
	}
	a := sortedEntries(confirmed)
	b := sortedEntries(attested)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedEntries(in []trademsg.CrcEntry) []trademsg.CrcEntry {
	out := copyEntries(in)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Data != out[j].Data {
			return out[i].Data < out[j].Data
		}
		return out[i].Crc < out[j].Crc
	})
	return out
}

// --- terminal saga status ------------------------------------------------------
//
// Everything below is driven by the DURABLE settlement record, not by the live
// room. The room is process-local in-memory state and the saga is not, so a
// restart between submission and the terminal status leaves a trade that has
// executed with no room to close it from — which is exactly the hole this
// record fills (see the settlement package).
//
// The record is also the ARBITER: whoever deletes it owns the outcome. That
// single atomic delete replaces what was previously a room compare-and-set, and
// unlike the room it works after a restart, when there is no room left to race
// over.

// SettlementSucceeded records the trade and closes both dialogs (FR-5.6,
// FR-7.1). Per design §6.4 it runs ONLY on terminal saga success, so the meso
// award has already reached atlas-character and the client's own figure is
// current.
//
// settlementId is the saga's transaction id — the only identity that survives a
// restart, and the one the status event carries.
func (p *ProcessorImpl) SettlementSucceeded(txId uuid.UUID, settlementId uuid.UUID) error {
	return p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.completeSettlement(mb, txId, settlementId, true, "")
	})
}

// SettlementFailed closes both dialogs with LEAVE 8 after the settlement saga
// reported terminal failure. No ledger row is written: a failed trade is
// observable through logs and metrics only (FR-7.3).
func (p *ProcessorImpl) SettlementFailed(txId uuid.UUID, settlementId uuid.UUID, reason string) error {
	return p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.completeSettlement(mb, txId, settlementId, false, reason)
	})
}

// completeSettlement finishes one settlement, with or without a live room.
//
// Order is load-bearing:
//
//  1. The LEDGER is written first, on success. Record is idempotent per
//     settlement transaction, so a delivery that goes on to lose the arbiter
//     has re-read the same row rather than written a second one — whereas
//     writing after the claim would make the audit record conditional on
//     winning a race it has no business depending on.
//  2. The RECORD is deleted, and its rows-affected decides the winner. Two
//     concurrent deliveries serialise on that delete, so exactly one proceeds
//     to emit.
//  3. The ROOM, if this process still has one, is removed.
//  4. The HOLDS are cancelled from the record — including on success, because
//     TradeSettlementItem carries no reservation id and atlas-inventory's
//     Release does not clear the reservation registry
//     (compartment/processor.go:1767-1855), so the hold would otherwise sit on
//     the giver's now-emptied slot for the rest of its 300s TTL.
//  5. The client-visible outcome is emitted, built entirely from the record.
func (p *ProcessorImpl) completeSettlement(mb *message.Buffer, txId uuid.UUID, settlementId uuid.UUID, success bool, reason string) error {
	sp := settlement.NewProcessor(p.l, p.ctx, p.db)
	s, err := sp.GetByTransactionId(settlementId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.Debugf("Terminal status for settlement [%s], which is already resolved. Ignoring.", settlementId.String())
			return nil
		}
		return err
	}

	var entryId uuid.UUID
	if success {
		entry, rerr := ledger.NewProcessor(p.l, p.ctx, p.db).Record(ledgerEntryFor(s))
		if rerr != nil {
			return rerr
		}
		entryId = entry.Id()
	}

	won, err := sp.Resolve(settlementId)
	if err != nil {
		return err
	}
	if !won {
		p.l.Debugf("Settlement [%s] was resolved by another delivery. Ignoring this one.", settlementId.String())
		return nil
	}

	if room, ok := p.reg.GetBySettlement(p.t, settlementId); ok {
		p.reg.RemoveIf(p.t, room.Id(), settling)
		p.timers.Cancel(p.t, room.Id())
	}

	if err = p.emitStagedReleases(mb, releasesFor(s)); err != nil {
		return err
	}

	if !success {
		recordSettlementFailed(p.t, sagaFailedReason)
		recordCancelled(p.t, ReasonTradeFailed)
		p.l.WithFields(p.settlementFields(s)).Warnf("Settlement [%s] failed: [%s].", settlementId.String(), reason)
		// Both sides are told, resolved from the RECORD. The FAILED event names
		// one character — the failed expanded step's — and that is not a role,
		// so it is never used to pick a side.
		return mb.Put(trademsg.EnvEventTopicStatus, recordCancelledProvider(txId, s, ReasonTradeFailed))
	}

	var taxed uint32
	for _, side := range s.Sides() {
		taxed += side.MesoTax()
	}
	recordSettled(p.t, taxed)
	p.l.WithFields(p.settlementFields(s)).WithField("ledger_entry_id", entryId.String()).Infof("Settlement [%s] settled.", settlementId.String())
	return mb.Put(trademsg.EnvEventTopicStatus, recordSettledProvider(txId, s, entryId))
}

// releasesFor returns the reservation cancels a settlement record owes.
//
// The slots come from the RECORD, which stored them already re-resolved at
// submission time, so no compartment re-read is needed — and on the success
// path none would help anyway: the asset has been released, so a lookup by
// asset id finds nothing and falls back to the recorded slot regardless.
func releasesFor(s settlement.Model) []stagedRelease {
	out := make([]stagedRelease, 0)
	for _, side := range s.Sides() {
		for _, i := range side.Items() {
			out = append(out, stagedRelease{
				reservationId: i.ReservationId(),
				characterId:   side.CharacterId(),
				inventoryType: i.InventoryType(),
				sourceSlot:    i.SourceSlot(),
			})
		}
	}
	return out
}

// ledgerEntryFor projects the durable settlement record onto the ledger row.
// The settlement saga's transaction id is the entry's, which is what makes the
// write idempotent per settlement (FR-5.7).
//
// Items carry the asset id but no referenceId: the record holds the asset's
// identity, and the equip/pet/cash reference lives on the asset in
// atlas-inventory, which has already released it by the time this runs.
func ledgerEntryFor(s settlement.Model) ledger.Model {
	b := ledger.NewBuilder(s.TransactionId(), s.Field(), s.RoomType())
	for _, side := range s.Sides() {
		items := make([]ledger.Item, 0, len(side.Items()))
		for _, i := range side.Items() {
			assetId := i.AssetId()
			items = append(items, ledger.NewItem(i.TemplateId(), i.Quantity(), &assetId, nil))
		}
		b.AddSide(side.CharacterId(), side.CharacterName(), side.MesoStaged(), side.MesoTax(), side.MesoDelivered(), items)
	}
	return b.Build()
}

// settlementRecordFor projects a room that has just won the SETTLING transition
// onto the durable record submitted alongside its saga.
func settlementRecordFor(settlementId uuid.UUID, room Room) settlement.Model {
	b := settlement.NewBuilder(settlementId, room.Id(), room.Handle(), room.RoomType(), room.Field(), room.OwnerId(), room.VisitorId())
	for _, pt := range room.OrderedParticipants() {
		items := make([]settlement.Item, 0, len(pt.Items()))
		for _, i := range pt.Items() {
			items = append(items, settlement.NewItem(i.ReservationId(), i.InventoryType(), i.SourceSlot(), i.AssetId(), i.TemplateId(), i.Quantity()))
		}
		b.AddSide(pt.Position(), pt.CharacterId(), pt.Name(), pt.MesoStaged(), pt.MesoTax(), pt.MesoDelivered(), items)
	}
	return b.Build()
}

// settlementFields is the structured-log context for a settlement that may no
// longer have a room — the restart case, where roomFields has nothing to read.
func (p *ProcessorImpl) settlementFields(s settlement.Model) logrus.Fields {
	f := logrus.Fields{
		"tenant_id":     p.t.Id().String(),
		"room_id":       s.RoomId().String(),
		"settlement_id": s.TransactionId().String(),
		"owner_id":      uint32(s.OwnerId()),
		"visitor_id":    uint32(s.VisitorId()),
	}
	return f
}

// --- startup reconciliation ------------------------------------------------------

// Reconcile completes every settlement this service submitted but never saw the
// outcome of — the trades a restart would otherwise have lost (FR-7.1).
//
// It runs at boot with NO tenant in context, so it enumerates the unfinished
// records across every tenant and restores each row's own tenant before doing
// anything with it. A failure for one tenant does not stop the others, and no
// failure stops the service: the caller runs this off the request path.
func Reconcile(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) error {
	records, err := settlement.Unresolved(ctx, db)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{})
	var failures int
	for _, s := range records {
		if _, ok := seen[s.TenantId()]; ok {
			continue
		}
		seen[s.TenantId()] = struct{}{}
		t, terr := s.Tenant()
		if terr != nil {
			l.WithError(terr).Errorf("Unable to restore tenant [%s] for settlement reconciliation. Its unfinished settlements are left for the next boot.", s.TenantId().String())
			failures++
			continue
		}
		if rerr := NewProcessor(l, tenant.WithContext(ctx, t), db).ReconcileSettlements(); rerr != nil {
			l.WithError(rerr).Errorf("Unable to reconcile settlements for tenant [%s].", t.Id().String())
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("settlement reconciliation failed for %d tenant(s)", failures)
	}
	return nil
}

// ReconcileSettlements completes this tenant's unfinished settlements by asking
// atlas-saga-orchestrator what became of each saga.
//
// It is safe to run repeatedly: a completed settlement's record is deleted, and
// ledger.Record is idempotent per transaction id, so a second pass over the same
// row can neither double-credit nor double-emit.
//
// An UNKNOWN outcome — the orchestrator unreachable, or the saga not yet
// consumed and therefore a 404 — leaves the record exactly where it is. It is
// never read as failure: a trade that may have executed must not be reported to
// the players as unsuccessful, and the live status consumer will resolve it the
// moment the terminal event arrives.
func (p *ProcessorImpl) ReconcileSettlements() error {
	records, err := settlement.NewProcessor(p.l, p.ctx, p.db).Unresolved()
	if err != nil {
		return err
	}
	for _, s := range records {
		outcome, oerr := p.sagad.Outcome(s.TransactionId())
		if oerr != nil {
			p.l.WithError(oerr).WithFields(p.settlementFields(s)).Warnf("Unable to read the outcome of settlement [%s]. Leaving it unresolved rather than guessing.", s.TransactionId().String())
			continue
		}
		switch outcome {
		case sagadata.OutcomeSucceeded:
			p.l.WithFields(p.settlementFields(s)).Infof("Reconciling settlement [%s]: the saga completed.", s.TransactionId().String())
			if cerr := p.SettlementSucceeded(uuid.New(), s.TransactionId()); cerr != nil {
				return cerr
			}
		case sagadata.OutcomeFailed:
			p.l.WithFields(p.settlementFields(s)).Infof("Reconciling settlement [%s]: the saga failed.", s.TransactionId().String())
			if cerr := p.SettlementFailed(uuid.New(), s.TransactionId(), sagaFailedReason); cerr != nil {
				return cerr
			}
		default:
			p.l.WithFields(p.settlementFields(s)).Infof("Settlement [%s] is still running; leaving it for the live status event.", s.TransactionId().String())
		}
	}
	return nil
}

// teardownRoom ends a room that is NOT settling: it releases both sides' holds
// and tells the clients why. characterId names whose action triggered it; for a
// settlement refusal there is no such actor, so callers pass the owner.
//
// The removal is a COMPARE-AND-SET, and that is load-bearing rather than
// defensive. Every caller read the room before the fallible slot re-resolution
// below, and a settlement can win the race to SETTLING inside that window —
// from the attestation deadline's own goroutine, which no Kafka partition
// ordering serialises against the teardown consumer. An unconditional removal
// there would cancel the holds the in-flight saga is about to consume AND
// delete the room its terminal status must find, so the swap would execute with
// no ledger row and no SETTLED while both clients had already seen LEAVE 2.
//
// Losing the claim is not an error: FR-6.5 says settlement wins and the client
// is reconciled by the settlement result.
func (p *ProcessorImpl) teardownRoom(mb *message.Buffer, txId uuid.UUID, room Room, characterId character.Id, reason string) error {
	claimed, ok := p.claimRoom(room, notSettling)
	if !ok {
		p.l.Infof("Teardown [%s] of trade room [%s] lost to its settlement. Ignoring: the saga's terminal status produces this room's LEAVE.", reason, room.Id().String())
		return nil
	}
	if err := p.emitStagedReleases(mb, claimed.releases); err != nil {
		return err
	}
	recordCancelled(p.t, reason)
	return mb.Put(trademsg.EnvEventTopicStatus, cancelledProvider(txId, claimed.room, characterId, reason))
}

// claimedRoom is a room this command has exclusively ended, together with the
// reservation cancels it owes.
type claimedRoom struct {
	room     Room
	releases []stagedRelease
}

// notSettling claims a room no settlement has taken over.
func notSettling(r Room) bool { return r.State() != StateSettling }

// settling claims a room whose settlement saga has reached a terminal status.
func settling(r Room) bool { return r.State() == StateSettling }

// claimRoom resolves the room's reservation cancels and then removes it, but
// only if `claim` still accepts the state it is in.
//
// The REST reads happen BEFORE the removal, which is emit's contract (the
// registry is in-memory and a rolled-back transaction does not restore a room),
// and the removal is atomic with the state test, which is what makes two
// concurrent enders mutually exclusive. Anything staged inside that window is
// picked up by withLateStages, so a hold filed late is still cancelled.
func (p *ProcessorImpl) claimRoom(room Room, claim func(Room) bool) (claimedRoom, bool) {
	releases := p.resolveStagedReleases(room)
	claimedModel, ok := p.reg.RemoveIf(p.t, room.Id(), claim)
	if !ok {
		return claimedRoom{}, false
	}
	p.timers.Cancel(p.t, claimedModel.Id())
	return claimedRoom{room: claimedModel, releases: withLateStages(releases, claimedModel)}, true
}

// roomFields is the structured-log context design §12 asks for at every state
// transition: tenant, room and both characters.
func (p *ProcessorImpl) roomFields(room Room) logrus.Fields {
	return logrus.Fields{
		"tenant_id":    p.t.Id().String(),
		"room_id":      room.Id().String(),
		"owner_id":     uint32(room.OwnerId()),
		"visitor_id":   uint32(room.VisitorId()),
		"room_state":   string(room.State()),
		"staged_items": stagedItemCount(room),
	}
}

func stagedItemCount(room Room) int {
	n := 0
	for _, pt := range room.Participants() {
		n += len(pt.Items())
	}
	return n
}
