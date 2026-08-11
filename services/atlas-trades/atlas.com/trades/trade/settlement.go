package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/escrow"
	"atlas-trades/kafka/message"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// sagaFailedReason labels the settlement-failure metric when the orchestrator,
// not a pre-check, refused the trade. Pre-check refusals carry their own
// leaveReason key.
const sagaFailedReason = "SAGA_FAILED"

// submitFailedReason labels the settlement-failure metric when the settlement
// never reached the orchestrator at all — the command that would have submitted
// it failed after the room had already been moved to SETTLING.
const submitFailedReason = "SUBMIT_FAILED"

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
	// ever reaches the clients, so no attestation can arrive and nothing else
	// settles it, leaving both sides' staged assets in escrow until some other
	// teardown trigger fires or the process restarts. confirm therefore reports
	// the room to arm as soon as the swap lands, before it buffers anything.
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
	var settled uuid.UUID
	err := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		var aerr error
		settled, aerr = txp.attest(mb, txId, characterId, entries)
		return aerr
	})
	return p.recoverAbandonedSettlement(txId, settled, err)
}

// attest returns the id of the room it drove to SETTLING, or uuid.Nil.
func (p *ProcessorImpl) attest(mb *message.Buffer, txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) (uuid.UUID, error) {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued TRANSACTION without a trade room. Dropping.", characterId)
		return uuid.Nil, nil
	}
	if room.State() != StateAwaitingAttestation {
		p.l.Debugf("Character [%d] issued TRANSACTION against trade room [%s] in state [%s]. Dropping.", characterId, room.Id().String(), room.State())
		return uuid.Nil, nil
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
		return uuid.Nil, nil
	}
	if !updated.BothAttested() {
		return uuid.Nil, nil
	}
	return p.settle(mb, txId, updated)
}

// ExpireAttestation settles a room whose attestation deadline lapsed, using
// whatever attestation arrived (design §3.1). The side that never replied is
// simply not CRC-checked: its TRADE_CONFIRM list is the only evidence there is,
// and comparing that list to itself would be a tautology.
func (p *ProcessorImpl) ExpireAttestation(txId uuid.UUID, roomId uuid.UUID) error {
	var settled uuid.UUID
	err := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		var eerr error
		settled, eerr = txp.expireAttestation(mb, txId, roomId)
		return eerr
	})
	return p.recoverAbandonedSettlement(txId, settled, err)
}

// expireAttestation returns the id of the room it drove to SETTLING, or
// uuid.Nil.
func (p *ProcessorImpl) expireAttestation(mb *message.Buffer, txId uuid.UUID, roomId uuid.UUID) (uuid.UUID, error) {
	room, ok := p.reg.Get(p.t, roomId)
	if !ok {
		p.l.Debugf("Attestation deadline for trade room [%s] fired after the room was gone. Ignoring.", roomId.String())
		return uuid.Nil, nil
	}
	if room.State() != StateAwaitingAttestation {
		p.l.Debugf("Attestation deadline for trade room [%s] fired in state [%s]. Ignoring.", roomId.String(), room.State())
		return uuid.Nil, nil
	}
	p.l.WithFields(p.roomFields(room)).Infof("Attestation deadline lapsed for trade room [%s]; settling on the confirm lists.", roomId.String())
	return p.settle(mb, txId, room)
}

// recoverAbandonedSettlement undoes a SETTLING transition whose command then
// failed.
//
// The registry swap to SETTLING is IN-MEMORY and is not rolled back by the
// enclosing transaction (see emit), but everything the swap was made for is:
// a failed command publishes no saga and writes no durable record. The room
// would then be stuck in a state nothing can act on — teardownCharacter refuses
// a settling room (FR-6.5), refreshReservations skips it, its attestation
// deadline has already been cancelled, and the reconciler has no record to find
// it by. Both dialogs would stay open for the rest of the process's life.
//
// The recovery therefore CLOSES the trade rather than reverting it to
// AWAITING_ATTESTATION: both sides have already attested, so nothing would ever
// drive settlement again, and an open dialog that can never complete is worse
// than a faithful LEAVE 8. Nothing moved — the saga was never published — so
// there is nothing to compensate.
//
// It runs in its OWN transaction, because the one that failed is already
// rolled back and anything buffered into it is gone.
func (p *ProcessorImpl) recoverAbandonedSettlement(txId uuid.UUID, roomId uuid.UUID, cause error) error {
	if cause == nil || roomId == uuid.Nil {
		return cause
	}
	p.l.WithError(cause).Errorf("Trade room [%s] reached SETTLING but its command failed; no saga was submitted. Closing the trade rather than leaving the room wedged.", roomId.String())
	if err := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.abandonSettlement(mb, txId, roomId)
	}); err != nil {
		// Second-order failure: the room stays SETTLING and the dialogs stay
		// open. Loud, because nothing downstream will retry it.
		p.l.WithError(err).Errorf("Unable to close trade room [%s] after its settlement failed to submit. The room is left settling.", roomId.String())
	}
	return cause
}

func (p *ProcessorImpl) abandonSettlement(mb *message.Buffer, txId uuid.UUID, roomId uuid.UUID) error {
	room, ok := p.reg.Get(p.t, roomId)
	if !ok {
		return nil
	}
	claimed, ok := p.claimRoom(room, settling)
	if !ok {
		// Something else already ended it — a terminal status that raced in, or
		// a previous recovery attempt.
		return nil
	}
	if err := p.emitUnwind(mb, claimed.room); err != nil {
		return err
	}
	recordSettlementFailed(p.t, submitFailedReason)
	recordCancelled(p.t, ReasonTradeFailed)
	return mb.Put(trademsg.EnvEventTopicStatus, cancelledProvider(txId, claimed.room, claimed.room.OwnerId(), ReasonTradeFailed))
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
//
// It returns the id of the room it moved to SETTLING, so the caller can undo
// that in-memory transition if the command it belongs to then fails — see
// recoverAbandonedSettlement. Every other outcome returns uuid.Nil: a refusal
// tears the room down itself, and a lost compare-and-set changed nothing.
func (p *ProcessorImpl) settle(mb *message.Buffer, txId uuid.UUID, room Room) (uuid.UUID, error) {
	if len(room.Participants()) != 2 {
		p.l.Errorf("Trade room [%s] reached settlement with [%d] participants. Tearing it down.", room.Id().String(), len(room.Participants()))
		return uuid.Nil, p.teardownRoom(mb, txId, room, room.OwnerId(), ReasonTradeFailed)
	}
	cfg := p.cfg.Get(p.l, p.ctx)
	cache := p.newCompartmentCache()

	// No slot re-resolution. Under reserve-at-staging a staged item was still in
	// its owner's inventory, so it could be dragged to a different slot between
	// the stage and the settlement and had to be re-resolved by asset id before
	// the payload was built. Escrow removes the possibility rather than the
	// need: the asset left the compartment when it was staged, so there is no
	// slot for it to move between, and the escrow row is immutable once written
	// (design §5A.10).
	resolved := room
	ordered := resolved.OrderedParticipants()
	var taxes [2]mesoSplit
	for i, pt := range ordered {
		tax, delivered := configuration.Tax(cfg, pt.MesoStaged())
		taxes[i] = mesoSplit{tax: tax, delivered: delivered}
	}

	if reason, ok := p.preCheck(cache, ordered, taxes); !ok {
		p.l.WithFields(p.roomFields(resolved)).Infof("Refusing to settle trade room [%s]: [%s].", resolved.Id().String(), reason)
		recordSettlementFailed(p.t, reason)
		return uuid.Nil, p.teardownRoom(mb, txId, resolved, resolved.OwnerId(), reason)
	}

	settlementId := uuid.New()
	// Compare-and-set. Another goroutine that already drove this room to
	// SETTLING wins, and this attempt becomes a no-op rather than a second saga
	// (design §12). The resolved tax split is written back in the same step, so
	// the payload built below and the ledger row written at terminal success are
	// derived from one and the same room.
	updated, err := p.reg.Update(p.t, resolved.Id(), func(cur Room) (Room, error) {
		if cur.State() != StateAwaitingAttestation {
			return Room{}, ErrRoomFrozen
		}
		next := cur
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
		return uuid.Nil, nil
	}

	p.timers.Cancel(p.t, updated.Id())

	// The DURABLE record is written in the SAME transaction that enqueues the
	// saga command, so submission and the record cannot diverge: either both
	// commit or neither does. Without it a restart before the terminal status
	// loses the room, and with it the trade that has already executed — no
	// ledger row, no SETTLED (FR-7.1).
	// From here on the room is SETTLING in memory and the swap will NOT be
	// rolled back with the transaction, so every remaining failure is returned
	// ALONGSIDE the room id: the caller closes the trade rather than leaving a
	// room nothing can act on.
	if _, err = settlement.NewProcessor(p.l, p.ctx, p.db).Submit(settlementRecordFor(settlementId, updated)); err != nil {
		return updated.Id(), err
	}

	p.l.WithFields(p.roomFields(updated)).WithField("settlement_id", settlementId.String()).Infof("Submitting settlement for trade room [%s].", updated.Id().String())
	payload, err := p.settlementPayload(settlementId, updated)
	if err != nil {
		return updated.Id(), err
	}
	if err = p.sgp.Settle(mb)(settlementId, payload); err != nil {
		return updated.Id(), err
	}
	return updated.Id(), nil
}

// mesoSplit is one side's resolved tax outcome: tax is destroyed, delivered
// goes to the counterparty.
type mesoSplit struct {
	tax       uint32
	delivered uint32
}

// settlementPayload projects the room onto the composite the orchestrator
// expands. Sides are ordered by seat, which the [2]TradeSettlementSide array
// requires; that order carries no role meaning (see Room.OrderedParticipants).
//
// The ITEMS come from the escrow store, not from the room. The room knows which
// escrow rows a side staged, but only the row itself carries the stat snapshot
// the orchestrator needs — it can no longer look the asset up, because the asset
// is not in anybody's compartment any more (design §5A.7). The room is still
// what decides side membership and the meso arithmetic.
//
// A staged item with no escrow row is an error, not a value to skip. It means
// the row was removed under a settlement that is already committing, and
// dropping it silently would settle the trade minus one item — the giver loses
// it and the receiver never gets it.
//
// The MESO check is the same guarantee for a balance, and it is why the room's
// figure is not trusted on its own. pt.MesoStaged() only advances when a stake
// RESOLVES, so a stake still in flight makes the room disagree with custody in
// whichever direction the stake was moving:
//
//   - a reduction in flight means the giver still has less escrowed than the
//     room believes, and delivering the room's figure MINTS the difference;
//   - a raise in flight means the debit lands after the room has been settled
//     and discharged, and the difference is DESTROYED.
//
// Which one happens is pure timing. An unmodified client can reach here in that
// state — CTradingRoomDlg::Trade sends TRADE_CONFIRM with no CanSendExclRequest
// gate, so nothing orders a confirm after the staging round trip it raced.
//
// The check lives HERE rather than at CONFIRM deliberately. That same client
// sets its own confirmed flag and disables its trade buttons before sending, and
// no server packet re-enables them, so refusing the confirm would wedge the
// dialog. Failing the settlement instead unwinds it, and both sides get
// everything back — the same outcome the item branch below already produces.
func (p *ProcessorImpl) settlementPayload(settlementId uuid.UUID, room Room) (sharedsaga.TradeSettlementPayload, error) {
	rows, err := p.esc.ItemsByRoom(room.Id())
	if err != nil {
		return sharedsaga.TradeSettlementPayload{}, err
	}
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	byEscrow := make(map[uuid.UUID]escrow.ItemModel, len(rows))
	for _, r := range rows {
		byEscrow[r.Id()] = r
	}

	payload := sharedsaga.TradeSettlementPayload{
		TransactionId: settlementId,
		WorldId:       room.Field().WorldId(),
		ChannelId:     room.Field().ChannelId(),
		RoomType:      room.RoomType(),
	}
	for i, pt := range room.OrderedParticipants() {
		if err = p.assertMesoCustodyAgrees(ep, room.Id(), pt); err != nil {
			return sharedsaga.TradeSettlementPayload{}, err
		}
		side := sharedsaga.TradeSettlementSide{
			CharacterId:   pt.CharacterId(),
			MesoStaged:    pt.MesoStaged(),
			MesoTax:       pt.MesoTax(),
			MesoDelivered: pt.MesoDelivered(),
		}
		for _, it := range pt.Items() {
			row, ok := byEscrow[it.EscrowId()]
			if !ok {
				return sharedsaga.TradeSettlementPayload{}, fmt.Errorf("staged item in trade slot %d of character %d has no escrow row %s", it.TradeSlot(), pt.CharacterId(), it.EscrowId())
			}
			side.Items = append(side.Items, escrowItemPayload(row))
		}
		payload.Sides[i] = side
	}
	return payload, nil
}

// assertMesoCustodyAgrees refuses to settle a side whose meso custody does not
// match what the room is about to deliver on its behalf.
//
// Two conditions, and both are about the same thing — the room's figure being a
// mirror rather than the truth:
//
//   - nothing may still be IN FLIGHT, because an unresolved stake means the
//     committed total is still moving;
//   - the committed total must EQUAL what the room thinks is staged, which
//     catches any other way the two could have diverged.
//
// Erroring is the point. The caller turns it into a failed settlement, which
// unwinds and returns everything, rather than delivering a figure nobody holds.
func (p *ProcessorImpl) assertMesoCustodyAgrees(ep escrow.Processor, roomId uuid.UUID, pt Participant) error {
	inFlight, err := ep.InFlightMesoDelta(roomId, pt.CharacterId())
	if err != nil {
		return err
	}
	if inFlight != 0 {
		return fmt.Errorf("character %d has a meso stake of %d still in flight; settling now would deliver a figure their escrow does not hold", pt.CharacterId(), inFlight)
	}
	committed, _, err := escrow.MesoByOwner(p.db, p.t.Id())(roomId, pt.CharacterId())
	if err != nil {
		return err
	}
	if committed != int64(pt.MesoStaged()) {
		return fmt.Errorf("character %d has %d meso escrowed but the room is settling %d on their behalf", pt.CharacterId(), committed, pt.MesoStaged())
	}
	return nil
}

// escrowItemPayload projects one custody row onto the saga's item view. It is
// shared by settlement and unwind: both hand the orchestrator the same snapshot,
// and they differ only in where the item is going.
func escrowItemPayload(r escrow.ItemModel) sharedsaga.TradeEscrowItem {
	return sharedsaga.TradeEscrowItem{
		EscrowId:      r.Id(),
		InventoryType: r.SourceInventoryType(),
		AssetId:       r.AssetId(),
		Snapshot:      r.Snapshot(),
	}
}

// claimItemsForReturn narrows a set of escrow rows to the ones THIS caller has
// won the exclusive right to return.
//
// Every path that submits a trade_unwind goes through it, because the paths are
// not mutually exclusive: a teardown and an orphaned stage's terminal status can
// both be looking at the same row, and a failed settlement's unwind and the boot
// sweep can both be looking at rows a previous pass already took. The claim is
// the only thing that decides between them — nothing downstream dedupes, so a
// row returned twice is granted to its owner twice (see
// escrow.ClaimItemForReturn).
func (p *ProcessorImpl) claimItemsForReturn(txId uuid.UUID, items []escrow.ItemModel) ([]escrow.ItemModel, error) {
	won := make([]escrow.ItemModel, 0, len(items))
	for _, r := range items {
		ok, err := p.esc.ClaimForReturn(r.Id(), txId)
		if err != nil {
			return nil, err
		}
		if !ok {
			p.l.Debugf("Trade escrow row [%s] is already being returned by another path. Leaving it out of this unwind.", r.Id().String())
			continue
		}
		won = append(won, r)
	}
	return won, nil
}

// emitUnwind returns every escrowed asset and meso in a room to the people it
// came from (design §5A.8).
//
// It reads the ESCROW, never the room, because the room and the escrow can
// legitimately disagree in both directions: a stage whose accept_to_trade has
// not yet written its row has a dialog slot but nothing to return, and a row
// whose stage completed after the room was claimed has no slot but is exactly
// what would otherwise be stranded. Returning what is actually escrowed is right
// for both.
//
// What the escrow row does NOT tell us is whether somebody else is already
// returning it. A row is written by the custody consumer the moment
// accept_to_trade lands, which is many hops before atlas-trades hears the stage
// succeeded and clears the item's dialog-side `pending` flag; ItemsByRoom has no
// pending filter, so for that whole window this teardown and the stage's own
// late status (returnOrphanedStage) both see the row and both would submit an
// unwind for it. Hence the claim: only the rows this call wins go into the
// payload.
//
// A room with nothing escrowed — or nothing left unclaimed — submits nothing.
// Most cancelled trades are empty, and a saga per empty teardown would be pure
// noise.
func (p *ProcessorImpl) emitUnwind(mb *message.Buffer, room Room) error {
	roomId := room.Id()
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	items, err := p.esc.ItemsByRoom(room.Id())
	if err != nil {
		return err
	}
	mesos, err := p.esc.MesosByRoom(room.Id())
	if err != nil {
		return err
	}
	if len(items) == 0 && len(mesos) == 0 {
		return nil
	}
	// The unwind's id is minted BEFORE anything is claimed, because every claim
	// is stamped with it: that is what lets a FAILED unwind release exactly the
	// rows and restore exactly the amounts it took (see
	// escrow.ReleaseItemReturnClaims and escrow.RestoreMesoRefunds).
	unwindTxId := uuid.New()
	items, err = p.claimItemsForReturn(unwindTxId, items)
	if err != nil {
		return err
	}

	payload := sharedsaga.TradeUnwindPayload{TransactionId: unwindTxId}
	for _, r := range items {
		payload.Items = append(payload.Items, sharedsaga.TradeUnwindItem{
			OwnerId: r.OwnerId(),
			Item:    escrowItemPayload(r),
		})
	}
	for _, m := range mesos {
		// CLAIM rather than read. Two paths can each decide to refund this row
		// — a teardown here and the boot/ticker sweep — and building the
		// payload from a figure merely read let both submit a refund for the
		// same total under READ COMMITTED. The claim zeroes and takes it in one
		// locked step, so at most one of them carries it.
		amount, claimed, err := ep.ClaimMesoForReturn(roomId, m.OwnerId())
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		// See unwindStranded: the claim is destructive, so what it took is
		// recorded against this unwind in the same transaction.
		if err = ep.RecordMesoRefund(unwindTxId, roomId, m.OwnerId(), amount); err != nil {
			return err
		}
		payload.Mesos = append(payload.Mesos, sharedsaga.TradeUnwindMeso{
			CharacterId: m.OwnerId(),
			WorldId:     room.Field().WorldId(),
			ChannelId:   room.Field().ChannelId(),
			Amount:      uint32(amount),
		})
	}
	if len(payload.Items) == 0 && len(payload.Mesos) == 0 {
		return nil
	}
	if err := p.retireClaimedMesos(roomId, payload.Mesos); err != nil {
		return err
	}
	return p.sgp.Unwind(mb)(payload.TransactionId, payload)
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

	// There is deliberately no "are the staged assets still intact" check here.
	// Under reserve-at-staging it re-read each giver's compartment to confirm the
	// asset had not been moved or spent out from under a hold that could lapse —
	// the reservation itself was not observable, so the asset stood in for it.
	// Escrow makes the question unaskable and unnecessary at once: the asset is
	// in atlas-trades' own custody row, not in a compartment anyone can touch,
	// and custody does not expire (design §5A.10).

	// (3) The attestation matches what the counterparty staged.
	for i, side := range ordered {
		if !attestationMatches(side, ordered[1-i].Items()) {
			p.l.Warnf("Character [%d]'s TRANSACTION CRC list does not attest the items character [%d] staged.", side.CharacterId(), ordered[1-i].CharacterId())
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

// attestationMatches reports whether side's TRANSACTION CRC list attests the
// items `incoming` — the counterparty's staged contribution.
//
// THE CONFIRM AND THE ATTESTATION ARE NOT THE SAME LIST. Verified against the
// GMS v83 client:
//
//   - CTradingRoomDlg::Trade @0x7c39a0 (0x11 TRADE_CONFIRM) walks BOTH dialog
//     arrays — member 113 (this character's own staged items) and member 114
//     (the counterparty's) — so the confirm carries EVERY item in the window,
//     interleaved per trade slot.
//   - CTradingRoomDlg::OnTrade @0x7c20bc (0x14 TRANSACTION) walks member 114
//     ONLY, so the attestation carries just the items this character is about
//     to RECEIVE.
//
// Comparing the two directly refuses every trade in which both sides stage an
// item: the confirm holds n+m pairs and the attestation m. A one-sided trade
// hid it — the giver's attestation is empty (lenient, below) and the receiver's
// window holds a single item, so the two lists coincidentally matched.
//
// So the attestation is checked against the counterparty's contribution on two
// axes, both of which the server knows independently of the client:
//
//   - its `data` values must be exactly the counterparty's staged template ids,
//     compared as a MULTISET — CItemInfo::GetItemCRC keys on the template, so
//     two staged stacks of the same item legitimately produce two identical
//     pairs and a set would silently accept one; and
//   - every attested pair must also appear in this side's OWN confirm list,
//     which is what still catches a client reporting one CRC for an item at
//     confirm time and a different one at attestation time.
//
// An EMPTY attestation always matches, unchanged. That is not laxity: the CRC
// list is absent from the wire entirely on GMS <= v79 (the serverbound
// tradeCrcPresent gate, design §4.4), and the timeout path settles with no
// attestation at all — in both cases there is nothing to compare, and refusing
// would break every legacy version and every slow client. A client that wants
// to skip the check can only do so by sending nothing, which is
// indistinguishable from not replying, which the deadline already settles.
//
// Order is not significant on either axis: the client builds both lists by
// walking its dialog slots, so a reordering is not evidence of tampering.
func attestationMatches(side Participant, incoming []StagedItem) bool {
	attested := side.AttestEntries()
	if len(attested) == 0 {
		return true
	}
	if len(attested) != len(incoming) {
		return false
	}

	confirmed := make(map[trademsg.CrcEntry]int, len(attested))
	for _, e := range side.ConfirmEntries() {
		confirmed[e]++
	}
	wanted := make(map[item.Id]int, len(incoming))
	for _, it := range incoming {
		wanted[it.TemplateId()]++
	}

	for _, e := range attested {
		if confirmed[e] == 0 {
			return false
		}
		confirmed[e]--
		id := item.Id(e.Data)
		if wanted[id] == 0 {
			return false
		}
		wanted[id]--
	}
	return true
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
//  2. The ESCROW MESO is discharged, on SUCCESS ONLY, and still BEFORE the
//     arbiter — see dischargeSettledMesos for what a row that outlives the
//     record costs.
//  3. The RECORD is deleted, and its rows-affected decides the winner. Two
//     concurrent deliveries serialise on that delete, so exactly one proceeds
//     to emit.
//  4. The ROOM, if this process still has one, is removed.
//  5. The ESCROW is unwound, on FAILURE ONLY. A failed settlement has been
//     compensated by the orchestrator, which restored every custody row it had
//     released — so the items are back in escrow with no trade left to deliver
//     them, and somebody has to hand them back. On success the expanded
//     release_from_trade steps already emptied the escrow of ITEMS, and
//     unwinding would be a second delivery of assets that are no longer there.
//  6. The client-visible outcome is emitted, built entirely from the record.
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

		// BEFORE the arbiter, deliberately. The settlement record is the only
		// thing keeping this room out of the boot sweep — ReconcileAtBoot builds
		// its `owned` exclusion set from the unresolved records — so discharging
		// AFTER sp.Resolve would open a window in which the row exists with no
		// record shielding it. A crash inside that window leaves a room-less,
		// non-zero row, and the next boot refunds the giver a stake they have
		// already handed over and the receiver has already been paid. Running
		// first closes the window entirely: the row can only outlive the record
		// if it was never discharged, and it cannot be.
		//
		// A delivery that goes on to LOSE the arbitration has merely repeated a
		// discharge that is idempotent by construction — zeroing a row already at
		// zero, and deleting one already deleted.
		if derr := p.dischargeSettledMesos(s.RoomId()); derr != nil {
			return derr
		}
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

	if !success {
		// Driven from the RECORD's room id, not from a live room: a restart
		// between submission and this status leaves no room at all, and the
		// escrow rows still have to go home.
		if err = p.unwindRecord(mb, s); err != nil {
			return err
		}
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

// unwindRecord returns a failed settlement's escrow from the DURABLE record.
//
// It reads the escrow store fresh rather than replaying the record's own item
// list. The record is a snapshot taken at submission; the escrow is what exists
// now, after the orchestrator's compensation decided which rows it managed to
// restore. Refunding the snapshot would hand back items whose restore failed —
// creating them out of nothing.
//
// Each row is CLAIMED before it goes into the payload, for the same reason
// emitUnwind claims: a row another return path already took must not be handed
// back a second time.
func (p *ProcessorImpl) unwindRecord(mb *message.Buffer, s settlement.Model) error {
	roomId := s.RoomId()
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	items, err := p.esc.ItemsByRoom(s.RoomId())
	if err != nil {
		return err
	}
	mesos, err := p.esc.MesosByRoom(s.RoomId())
	if err != nil {
		return err
	}
	if len(items) == 0 && len(mesos) == 0 {
		return nil
	}
	// The unwind's id is minted BEFORE anything is claimed, because every claim
	// is stamped with it: that is what lets a FAILED unwind release exactly the
	// rows and restore exactly the amounts it took (see
	// escrow.ReleaseItemReturnClaims and escrow.RestoreMesoRefunds).
	unwindTxId := uuid.New()
	items, err = p.claimItemsForReturn(unwindTxId, items)
	if err != nil {
		return err
	}

	payload := sharedsaga.TradeUnwindPayload{TransactionId: unwindTxId}
	for _, r := range items {
		payload.Items = append(payload.Items, sharedsaga.TradeUnwindItem{
			OwnerId: r.OwnerId(),
			Item:    escrowItemPayload(r),
		})
	}
	for _, m := range mesos {
		// CLAIM rather than read. Two paths can each decide to refund this row
		// — a teardown here and the boot/ticker sweep — and building the
		// payload from a figure merely read let both submit a refund for the
		// same total under READ COMMITTED. The claim zeroes and takes it in one
		// locked step, so at most one of them carries it.
		amount, claimed, err := ep.ClaimMesoForReturn(roomId, m.OwnerId())
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		// See unwindStranded: the claim is destructive, so what it took is
		// recorded against this unwind in the same transaction.
		if err = ep.RecordMesoRefund(unwindTxId, roomId, m.OwnerId(), amount); err != nil {
			return err
		}
		payload.Mesos = append(payload.Mesos, sharedsaga.TradeUnwindMeso{
			CharacterId: m.OwnerId(),
			WorldId:     s.Field().WorldId(),
			ChannelId:   s.Field().ChannelId(),
			Amount:      uint32(amount),
		})
	}
	if len(payload.Items) == 0 && len(payload.Mesos) == 0 {
		return nil
	}
	if err := p.retireClaimedMesos(roomId, payload.Mesos); err != nil {
		return err
	}
	return p.sgp.Unwind(mb)(payload.TransactionId, payload)
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
			items = append(items, settlement.NewItem(i.EscrowId(), i.InventoryType(), i.SourceSlot(), i.AssetId(), i.TemplateId(), i.Quantity()))
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

// ReconcileAtBoot runs the two startup passes in the one order that is safe:
// settlements first, then stranded escrow (design §5A.9).
//
// The ordering is mandatory because a settlement still in flight legitimately
// owns escrow rows — its own release or unwind will consume them — so sweeping
// escrow first would hand the giver back an item the settlement then also
// delivers to the receiver, minting it.
//
// The exclusion set is captured BEFORE the settlement pass, not after. Reconcile
// deletes each record as it resolves it, so reading the unresolved set
// afterwards would return nothing and the sweep would treat rows belonging to a
// just-resolved settlement — whose release saga is still in flight, moments
// behind us in the same process — as stranded. Those are precisely the rows a
// double delivery would come from.
func ReconcileAtBoot(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) error {
	inFlight, err := settlement.Unresolved(ctx, db)
	if err != nil {
		return err
	}
	owned := make(map[uuid.UUID]struct{}, len(inFlight))
	for _, s := range inFlight {
		owned[s.RoomId()] = struct{}{}
	}

	rerr := Reconcile(l, ctx, db)
	if eerr := ReconcileEscrow(l, ctx, db, owned); eerr != nil {
		if rerr != nil {
			return fmt.Errorf("%w; and escrow reconciliation failed: %w", rerr, eerr)
		}
		return eerr
	}
	return rerr
}

// ReconcileEscrow returns every escrowed item and meso whose trade cannot
// possibly still be running (design §5A.9).
//
// At boot that is nearly all of them: the room registry is in memory, so a
// restart loses every live room, and an escrow row with no room is an asset
// nobody can reach — the player's item, held by a trade that no longer exists.
// Rows whose room is named in `owned` are skipped: a settlement still holds
// them.
//
// It runs with NO tenant in context and restores each row's own tenant, the same
// shape as Reconcile. A failure for one tenant does not stop the others.
func ReconcileEscrow(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, owned map[uuid.UUID]struct{}) error {
	items, err := escrow.AllItems(db)
	if err != nil {
		return err
	}
	mesos, err := escrow.AllMesos(db)
	if err != nil {
		return err
	}
	if len(items) == 0 && len(mesos) == 0 {
		return nil
	}

	// Grouped by (tenant, room): one unwind saga per stranded room, so a room
	// that held several items produces one saga rather than one per item.
	type roomKey struct {
		tenantId uuid.UUID
		roomId   uuid.UUID
	}
	rooms := make(map[roomKey]*strandedRoom)
	claim := func(k roomKey, t func() (tenant.Model, error)) *strandedRoom {
		if _, skip := owned[k.roomId]; skip {
			return nil
		}
		if r, ok := rooms[k]; ok {
			return r
		}
		tm, terr := t()
		if terr != nil {
			l.WithError(terr).Errorf("Unable to restore tenant [%s] for escrow reconciliation. Its stranded rows are left for the next boot.", k.tenantId.String())
			return nil
		}
		r := &strandedRoom{tenant: tm, roomId: k.roomId}
		rooms[k] = r
		return r
	}

	for _, i := range items {
		if r := claim(roomKey{i.TenantId(), i.RoomId()}, i.Tenant); r != nil {
			r.items = append(r.items, i)
		}
	}
	for _, m := range mesos {
		// Non-positive means nothing is owed: a zero row holds no custody, and a
		// negative one means more reduction has been confirmed than increase so
		// far (see MesoEntity on why the confirmed total is signed).
		if m.Amount() <= 0 {
			continue
		}
		if r := claim(roomKey{m.TenantId(), m.RoomId()}, m.Tenant); r != nil {
			r.mesos = append(r.mesos, m)
		}
	}

	var failures int
	for _, r := range rooms {
		l.Warnf("Trade room [%s] left [%d] item(s) and [%d] meso row(s) in escrow with no room to claim them. Returning them to their owners.", r.roomId.String(), len(r.items), len(r.mesos))
		p := NewProcessor(l, tenant.WithContext(ctx, r.tenant), db).(*ProcessorImpl)
		if uerr := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
			return txp.unwindStranded(mb, r)
		}); uerr != nil {
			l.WithError(uerr).Errorf("Unable to return the escrow of trade room [%s]. The rows are durable and are retried at the next boot.", r.roomId.String())
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("escrow reconciliation failed for %d room(s)", failures)
	}
	return nil
}

// retireClaimedMesos removes the rows an unwind has just CLAIMED and is about
// to refund, where nothing is left outstanding on them.
//
// Somebody has to retire them, because the unwind saga does not: expandTradeUnwind
// releases each ITEM (which deletes its custody row) but a meso leg is a bare
// award_mesos, so without this the row survives its own refund and the next
// sweep — a second teardown, or the boot pass — reads it again.
//
// It does NOT zero, because ClaimMesoForReturn already did, under a row lock, as
// the very act of taking the amount. Re-assigning zero here would be worse than
// redundant: a sibling stake committing its delta between the claim and this
// call would have that delta silently clobbered.
//
// The delete is CONDITIONAL (DeleteResolvedMeso) and that is load-bearing: a
// stake still in flight resolves against this row, and deleting it would strand
// a debit the player has already been charged with no record left to refund from
// (see resolveMesoStake's room-gone branch). Because the condition lives inside
// the DELETE's own WHERE clause, a stage arming a stake concurrently keeps its
// row.
func (p *ProcessorImpl) retireClaimedMesos(roomId uuid.UUID, mesos []sharedsaga.TradeUnwindMeso) error {
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	for _, m := range mesos {
		if _, err := ep.DeleteResolvedMeso(roomId, m.CharacterId); err != nil {
			return err
		}
	}
	return nil
}

// dischargeSettledMesos retires the room's escrow meso rows once its settlement
// has SUCCEEDED. It is the meso counterpart of the release_from_trade steps the
// expanded settlement saga already emits for every ITEM.
//
// Without it a successful trade MINTS meso. expandTradeSettlement's meso leg is
// a bare award_mesos CREDIT to the receiver; no step removes or zeroes the
// giver's escrow row. The row therefore survives the trade, and the moment
// sp.Resolve deletes the settlement record nothing names its room any more — so
// the next boot's ReconcileEscrow finds a room-less row carrying the full
// MesoStaged, misses ReconcileAtBoot's `owned` exclusion set, and refunds the
// giver a stake they have already handed over and the receiver has already been
// paid. That fires on EVERY successful settlement that staged meso, not on a
// race.
//
// It is driven from the DURABLE record's room id rather than a live room, for
// the same reason unwindRecord is: a restart between submission and the terminal
// status leaves no room at all, and the discharge still has to happen.
//
// The rows are read fresh rather than replayed off the record's MesoStaged
// figures, so a row a concurrent path already retired contributes nothing.
func (p *ProcessorImpl) dischargeSettledMesos(roomId uuid.UUID) error {
	mesos, err := p.esc.MesosByRoom(roomId)
	if err != nil {
		return err
	}
	owners := make([]character.Id, 0, len(mesos))
	for _, m := range mesos {
		owners = append(owners, m.OwnerId())
	}
	return p.retireEscrowMesos(roomId, owners)
}

// retireEscrowMesos is the zero-then-conditionally-delete both discharge paths
// share. It is one function rather than two because the guarantee is the same
// either way — the escrow this room recorded is settled, by refund or by
// delivery — and only the way its owners are named differs.
//
// The conditional delete is the load-bearing half: DeleteResolvedMeso refuses a
// row still carrying a pending stake, so the zeroing above it is what keeps a
// stale total out of the next sweep for exactly the rows that must survive.
func (p *ProcessorImpl) retireEscrowMesos(roomId uuid.UUID, owners []character.Id) error {
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	for _, o := range owners {
		if err := ep.UpsertMeso(roomId, o, 0); err != nil {
			return err
		}
		if _, err := ep.DeleteResolvedMeso(roomId, o); err != nil {
			return err
		}
	}
	return nil
}

// strandedRoom is one dead room's escrow, grouped for a single unwind.
type strandedRoom struct {
	tenant tenant.Model
	roomId uuid.UUID
	items  []escrow.ItemModel
	mesos  []escrow.MesoModel
}

// unwindStranded submits the return for one dead room.
//
// The meso legs need a world and channel to render their stat update, and the
// room that would have supplied them is gone, so each owner's CURRENT field is
// read instead. An owner who cannot be located is SKIPPED rather than defaulted
// to world 0 — a refund announced onto the wrong channel is worse than one that
// waits for the next boot, because the row is deleted either way.
func (p *ProcessorImpl) unwindStranded(mb *message.Buffer, r *strandedRoom) error {
	// A row already claimed by a return that is in flight is NOT stranded, it is
	// merely unfinished, and sweeping it into a second unwind would grant its
	// owner the item twice. The claim is the only durable record of that, which
	// is why it is a column rather than in-memory state: this sweep runs at boot,
	// with every room already lost.
	unwindTxId := uuid.New()
	items, err := p.claimItemsForReturn(unwindTxId, r.items)
	if err != nil {
		return err
	}

	payload := sharedsaga.TradeUnwindPayload{TransactionId: unwindTxId}
	for _, i := range items {
		payload.Items = append(payload.Items, sharedsaga.TradeUnwindItem{
			OwnerId: i.OwnerId(),
			Item:    escrowItemPayload(i),
		})
	}
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	for _, m := range r.mesos {
		// The field is resolved BEFORE the claim, and the order is
		// load-bearing: claiming zeroes the row, so a lookup that failed after
		// it would drop the only record that this meso is owed. Leaving the row
		// unclaimed hands it to the next sweep intact.
		f, ferr := p.locp.FieldOf(m.OwnerId())
		if ferr != nil {
			p.l.WithError(ferr).Errorf("Unable to locate character [%d] to refund [%d] escrowed meso from dead trade room [%s]. Left for the next boot.", m.OwnerId(), m.Amount(), r.roomId.String())
			continue
		}
		// See unwindRoom: claim rather than read, so a teardown racing this
		// sweep cannot refund the same total twice.
		amount, claimed, cerr := ep.ClaimMesoForReturn(r.roomId, m.OwnerId())
		if cerr != nil {
			return cerr
		}
		if !claimed {
			continue
		}
		// Durably name what this unwind took, in the same transaction that took
		// it. The claim is destructive — it zeroes the row — so without this a
		// failed unwind would destroy the meso with nothing left to restore it
		// from.
		if rerr := ep.RecordMesoRefund(unwindTxId, r.roomId, m.OwnerId(), amount); rerr != nil {
			return rerr
		}
		payload.Mesos = append(payload.Mesos, sharedsaga.TradeUnwindMeso{
			CharacterId: m.OwnerId(),
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      uint32(amount),
		})
	}
	if len(payload.Items) == 0 && len(payload.Mesos) == 0 {
		return nil
	}
	if err := p.retireClaimedMesos(r.roomId, payload.Mesos); err != nil {
		return err
	}
	return p.sgp.Unwind(mb)(payload.TransactionId, payload)
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
	if err := p.emitUnwind(mb, claimed.room); err != nil {
		return err
	}
	recordCancelled(p.t, reason)
	return mb.Put(trademsg.EnvEventTopicStatus, cancelledProvider(txId, claimed.room, characterId, reason))
}

// claimedRoom is a room this command has exclusively ended.
//
// It no longer carries a release list. Under reserve-at-staging the claim had to
// resolve, BEFORE the removal, which reservations it owed — the room was the
// only record of them, and removing it destroyed that record. The escrow store
// is durable and outlives the room, so the unwind can read it afterwards; the
// whole late-stage race the old resolve-then-claim dance existed to close
// disappears with it (design §5A.10).
type claimedRoom struct {
	room Room
}

// notSettling claims a room no settlement has taken over.
func notSettling(r Room) bool { return r.State() != StateSettling }

// settling claims a room whose settlement saga has reached a terminal status.
func settling(r Room) bool { return r.State() == StateSettling }

// claimRoom removes the room, but only if `claim` still accepts the state it is
// in. The test and the removal are atomic, which is what makes two concurrent
// enders mutually exclusive.
func (p *ProcessorImpl) claimRoom(room Room, claim func(Room) bool) (claimedRoom, bool) {
	claimedModel, ok := p.reg.RemoveIf(p.t, room.Id(), claim)
	if !ok {
		return claimedRoom{}, false
	}
	p.timers.Cancel(p.t, claimedModel.Id())
	return claimedRoom{room: claimedModel}, true
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

// UnwindFailed recovers from a trade_unwind that FAILED, and reports whether
// this transaction was an unwind at all.
//
// It exists because an unwind owns none of the id spaces the saga-status
// consumer probes — it is not a stage, not a meso stake, not a settlement — so
// its FAILED event fell through every one of them and was swallowed with a
// debug line. Nothing noticed, and the cost was total on both columns:
//
//   - ITEMS stayed latched by their return claim, which clears only on a
//     completed release. The boot sweep skips a latched row by design, so the
//     item sat intact in custody, owned by nobody and invisible to every path
//     that could have returned it.
//   - MESO was already zeroed, because claiming IS taking (ClaimMesoForReturn).
//     The row read zero, the refund never landed, and no record anywhere said
//     it had been owed.
//
// Recovery is therefore the exact inverse of what the claim did: release the
// item latches this transaction placed, and add back the amounts it recorded.
// Both are scoped to the transaction — rows another unwind is legitimately
// returning keep their claims — and both consume what they act on, so a
// redelivered failure is inert.
//
// Nothing is re-submitted here. The point is to put custody back into a state
// the ordinary paths can act on: the next teardown or boot sweep finds
// unlatched rows and a non-zero balance, and returns them.
func (p *ProcessorImpl) UnwindFailed(txId uuid.UUID, reason string) (bool, error) {
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)

	released, err := ep.ReleaseItemReturnClaims(txId)
	if err != nil {
		return false, err
	}
	restored, err := ep.RestoreMesoRefunds(txId)
	if err != nil {
		return false, err
	}
	if released == 0 && restored == 0 {
		return false, nil
	}

	p.l.Warnf("Trade unwind [%s] failed: [%s]. Released [%d] latched escrow item(s) and restored [%d] meso refund(s); the next sweep will return them.", txId.String(), reason, released, restored)
	return true, nil
}

// UnwindSucceeded discards the refund records of a completed trade_unwind and
// reports whether this transaction was an unwind.
//
// The meso reached the player, so there is nothing left to put back. Without
// this the records accumulate forever and a later redelivery of the FAILED
// event — which at-least-once delivery permits — would restore meso that has
// already been paid out, minting it.
//
// The item latches are deliberately NOT cleared: a completed unwind released
// those rows, and a released row is gone from the default scope entirely.
func (p *ProcessorImpl) UnwindSucceeded(txId uuid.UUID) (bool, error) {
	discarded, err := escrow.NewProcessor(p.l, p.ctx, p.db).DiscardMesoRefunds(txId)
	if err != nil {
		return false, err
	}
	return discarded > 0, nil
}
