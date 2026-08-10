package trade

import (
	"atlas-trades/compartment"
	"atlas-trades/configuration"
	characterdata "atlas-trades/data/character"
	inventorydata "atlas-trades/data/inventory"
	itemdata "atlas-trades/data/item"
	"atlas-trades/data/location"
	mapdata "atlas-trades/data/map"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/kafka/message"
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"
	sagaproducer "atlas-trades/saga"
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// enterError KEY strings. atlas-channel resolves each to the per-version
// numeric code through the tenant `enterError` table (DOM-25); the numbers in
// the trailing comments are the gms_v83 template's values and exist only to
// make the intent readable here.
const (
	errRoomClosed      = "ROOM_CLOSED"       // 1
	errOtherRequests   = "OTHER_REQUESTS"    // 3
	errNotWhenDead     = "NOT_WHEN_DEAD"     // 4
	errUnable          = "UNABLE"            // 6
	errTradeNotAllowed = "TRADE_NOT_ALLOWED" // 7
	errNotSameMap      = "NOT_SAME_MAP"      // 9
)

// leaveReason KEY strings for the trade family (design §4.3). They are exported
// because the teardown consumers pick which one a trigger carries and pass it to
// TeardownCharacter. The trailing numbers are the reference client's status
// bytes; the actual values come from the tenant `leaveReason` table.
// There is deliberately no TRADE_SUCCESS constant. A settled trade is announced
// as SETTLED, not as a cancellation reason: it is the only outcome that carries
// a ledger entry id, and atlas-channel resolves its own LEAVE 7 from that event.
const (
	ReasonTradeCancelled    = "TRADE_CANCELLED"     // 2
	ReasonTradeFailed       = "TRADE_FAILED"        // 8
	ReasonTradeCannotCarry  = "TRADE_CANNOT_CARRY"  // 9
	ReasonTradeDifferentMap = "TRADE_DIFFERENT_MAP" // 12
	ReasonTradeCrcFailed    = "TRADE_CRC_FAILED"    // 13
)

// inviteResult KEY strings. Derived from the GMS v83 client's
// CMiniRoomBaseDlg::OnInviteResultStatic (0x65E848), which decodes one byte and
// branches: 1 -> SP_366 "Unable to find the character" (no name follows);
// 2 -> SP_403 "%s is doing something else right now"; 3 -> SP_404 "%s have
// denied invitation"; 4 -> SP_405 "%s is currently not accepting any
// invitation". 0 renders nothing.
const (
	inviteResultCannotFind = "CANNOT_FIND_CHARACTER" // 1
	inviteResultBusy       = "BUSY"                  // 2
)

// maxStageableMeso bounds a single side's staged meso. atlas-saga-orchestrator's
// trade_settlement expansion refuses a payload whose MesoStaged or MesoDelivered
// exceeds math.MaxInt32, because AwardMesosPayload.Amount is a signed int32 and
// a larger value would wrap the giver's DEDUCTION into a credit
// (services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1429-1434).
// Staging is where a player can actually type a number, so the bound is enforced
// here too rather than being discovered at settlement, when the only remaining
// outcome is LEAVE 8.
const maxStageableMeso = math.MaxInt32

// characterProvider, mapProvider and locationProvider are the REST-client seams
// the validation ladder reads through, injected so tests can fake them.
type characterProvider interface {
	GetById(characterId character.Id) (characterdata.Model, error)
}

type mapProvider interface {
	FieldLimit(mapId _map.Id) (uint32, error)
}

type locationProvider interface {
	FieldOf(characterId character.Id) (field.Model, error)
}

// inventoryProvider reads the asset a PUT_ITEM addresses, and re-reads a whole
// compartment when a staged item's recorded slot has to be re-resolved against
// the asset's stable id (see resolveStagedSlot).
type inventoryProvider interface {
	AssetInSlot(characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) (inventorydata.Asset, error)
	GetCompartment(characterId character.Id, inventoryType inventory.Type) (inventorydata.Model, error)
}

// itemDataProvider reads the WZ facts staging and settlement need: the
// tradeBlock flag (FR-4.2) and the slotMax the free-slot pre-check counts
// stackable merges with (design §6.1 check 1).
type itemDataProvider interface {
	TradeBlock(inventoryType inventory.Type, templateId item.Id) (bool, error)
	SlotMax(inventoryType inventory.Type, templateId item.Id) (uint32, error)
}

// sagaOutcomeProvider reads what became of a submitted saga. Only startup
// reconciliation asks: the live path is driven by the status event.
type sagaOutcomeProvider interface {
	Outcome(transactionId uuid.UUID) (sagadata.Outcome, error)
}

// settlementSubmitter buffers the trade_settlement saga command. atlas-trades
// submits the COMPOSITE only; the orchestrator expands it (design §6.3).
type settlementSubmitter interface {
	Settle(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error
}

// reservationProducer issues the atlas-inventory reserve/cancel commands that
// implement the reserve-at-staging model (design §5.3).
type reservationProducer interface {
	RequestReserve(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, templateId item.Id, quantity asset.Quantity, expiry time.Duration) error
	CancelReservation(mb *message.Buffer) func(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) error
}

// configProvider resolves the request tenant's trade configuration.
// *configuration.Registry implements it.
type configProvider interface {
	Get(l logrus.FieldLogger, ctx context.Context) configuration.Model
}

// Processor owns every trade-room operation. REST handlers and Kafka consumers
// go through it rather than reaching into the registry directly (DOM-14).
type Processor interface {
	// RoomsForTenant returns every live room the request's tenant owns.
	RoomsForTenant() []Room

	// RoomById returns the room with the given id, scoped to the request's
	// tenant. A settled or cancelled room is gone, so a miss is reported rather
	// than a stale snapshot.
	RoomById(id uuid.UUID) (Room, bool)

	// RoomForCharacter returns the room the character occupies as either side.
	RoomForCharacter(characterId character.Id) (Room, bool)

	// RoomByHandle resolves a room from the uint32 wire serial the client's
	// invite and ENTER_ROOM carry.
	RoomByHandle(handle uint32) (Room, bool)

	// CreateRoom opens a solo trade room for characterId (FR-1.1).
	CreateRoom(txId uuid.UUID, f field.Model, characterId character.Id, roomType byte) error

	// Invite offers the caller's room to targetCharacterId (FR-2.1).
	Invite(txId uuid.UUID, f field.Model, characterId character.Id, targetCharacterId character.Id) error

	// DeclineInvite tears down the originator's pending room on a
	// client-originated decline, and retires the offer in atlas-invites (FR-2.5).
	DeclineInvite(txId uuid.UUID, characterId character.Id, originatorId character.Id) error

	// InviteRejected tears down the originator's pending room after
	// atlas-invites already retired the invite — a reject it processed, or an
	// expiry (FR-2.6).
	InviteRejected(txId uuid.UUID, characterId character.Id, originatorId character.Id) error

	// EnterRoom seats characterId in the room owning the given wire handle
	// (FR-1.5, FR-2.4). It admits ONLY the character the room's outstanding
	// invite named, and only when roomType matches the room's own kind — the
	// handle is public, so it is not on its own an admission ticket.
	EnterRoom(txId uuid.UUID, f field.Model, characterId character.Id, handle uint32, roomType byte) error

	// PutItem stages one item into the caller's side of the trade dialog
	// (FR-3.1..FR-3.3). inventoryType, sourceSlot, quantity and targetSlot are
	// the raw serverbound OperationTradePutItem fields; every one of them is
	// untrusted and validated here. A refused stage is SILENT to the client —
	// see checkRestrictions.
	PutItem(txId uuid.UUID, characterId character.Id, inventoryType byte, sourceSlot slot.Position, quantity uint16, targetSlot byte) error

	// Chat relays one line of trade-room chat to both sides (FR-8.1). The room
	// is resolved from the speaker's membership; a non-member is a silent no-op.
	Chat(txId uuid.UUID, characterId character.Id, text string) error

	// AddMeso sets the caller's staged meso to the ABSOLUTE amount their input
	// box carried (design §1.6). Signed because the serverbound codec is an
	// Encode4 of a signed int32 and a hostile client can send a negative.
	AddMeso(txId uuid.UUID, characterId character.Id, amount int32) error

	// Confirm records one side pressing Trade (FR-5.1). It freezes staging from
	// the FIRST confirm and, only once BOTH have confirmed, moves the room to
	// AWAITING_ATTESTATION and prompts both clients (design §6.2). entries is
	// the CRC list the payload carried, empty on the versions that have none.
	Confirm(txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) error

	// Attest records one side's automatic TRANSACTION reply to that prompt and,
	// once both have replied, runs the settlement pre-checks and submits the
	// saga.
	Attest(txId uuid.UUID, characterId character.Id, entries []trademsg.CrcEntry) error

	// ExpireAttestation settles a room whose attestation deadline lapsed, using
	// whatever attestation arrived (design §3.1).
	ExpireAttestation(txId uuid.UUID, roomId uuid.UUID) error

	// SettlementSucceeded writes the ledger row, releases both sides' holds and
	// closes both dialogs with LEAVE 7. Called only on terminal saga success.
	//
	// It is keyed by the SETTLEMENT saga's transaction id rather than by a room
	// id, because the room may not exist: a restart between submission and the
	// terminal status leaves only the durable settlement record, which this
	// path is driven from.
	SettlementSucceeded(txId uuid.UUID, settlementId uuid.UUID) error

	// SettlementFailed releases both sides' holds and closes both dialogs with
	// LEAVE 8. No ledger row is written (FR-7.3). Keyed by settlement id for
	// the same reason as SettlementSucceeded.
	SettlementFailed(txId uuid.UUID, settlementId uuid.UUID, reason string) error

	// ReconcileSettlements completes this tenant's settlements whose terminal
	// status was never seen — the trades a restart would otherwise lose.
	ReconcileSettlements() error

	// RefreshReservations re-files the inventory reservation of every staged
	// item in every live room, so a reservation TTL never expires under a trade
	// that is still open (design §5.3). Task 20 drives it from a ticker.
	//
	// It takes no transaction id: it emits no status event, only the
	// COMPARTMENT commands, which are keyed by the reservation handle they
	// re-file rather than by a command transaction.
	RefreshReservations() error

	// TeardownCharacter removes the character's room, unless it is already
	// settling (design §3.3, "cancel loses to settlement").
	TeardownCharacter(txId uuid.UUID, characterId character.Id, reason string) error
}

type ProcessorImpl struct {
	l      logrus.FieldLogger
	ctx    context.Context
	db     *gorm.DB
	t      tenant.Model
	reg    *Registry
	cfg    configProvider
	cp     characterProvider
	mp     mapProvider
	locp   locationProvider
	invp   inventoryProvider
	idp    itemDataProvider
	resp   reservationProducer
	sgp    settlementSubmitter
	sagad  sagaOutcomeProvider
	timers *attestationTimers
}

// NewProcessor resolves the tenant from ctx once; every registry read the
// processor issues is partitioned by that tenant.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		reg:   GetRegistry(),
		cfg:   configuration.GetRegistry(),
		cp:    characterdata.NewProcessor(l, ctx),
		mp:    mapdata.NewProcessor(l, ctx),
		locp:  location.NewProcessor(l, ctx),
		invp:  inventorydata.NewProcessor(l, ctx),
		idp:   itemdata.NewProcessor(l, ctx),
		resp:  compartment.NewProcessor(l, ctx),
		sgp:   sagaproducer.NewProcessor(l, ctx),
		sagad: sagadata.NewProcessor(l, ctx),
		// Process-wide: an attestation deadline armed by one command is
		// cancelled by another, which will be running on a different processor.
		timers: GetAttestationTimers(),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RoomsForTenant() []Room { return p.reg.All(p.t) }

func (p *ProcessorImpl) RoomById(id uuid.UUID) (Room, bool) { return p.reg.Get(p.t, id) }

func (p *ProcessorImpl) RoomForCharacter(characterId character.Id) (Room, bool) {
	return p.reg.GetByMember(p.t, characterId)
}

func (p *ProcessorImpl) RoomByHandle(handle uint32) (Room, bool) {
	return p.reg.GetByHandle(p.t, handle)
}

// emit runs a command's whole event batch through the transactional outbox
// (task-114): it opens one DB transaction, hands the closure a tx-scoped copy of
// the processor, and enqueues every buffered message into the outbox table
// inside that same transaction. A command's durable writes and all its status
// events therefore commit atomically and publish in buffer order.
//
// Registry mutations inside the closure are in-memory (the registry is never
// persisted, and atlas-trades runs replicas: 1 — design §9), so a rolled-back
// transaction does NOT roll a room back. Commands consequently do every fallible
// read before the registry swap.
func (p *ProcessorImpl) emit(f func(p *ProcessorImpl, mb *message.Buffer) error) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		txp := p.withTx(tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return f(txp, mb)
		})
	})
}

// withTx returns a shallow copy of the processor whose db handle is the given
// transaction.
func (p *ProcessorImpl) withTx(tx *gorm.DB) *ProcessorImpl {
	cp := *p
	cp.db = tx
	return &cp
}

// registryErrorCode maps a registry rejection to the enterError KEY the client
// should see. Every sentinel the registry can return is mapped: an unmapped one
// would degrade to a generic failure and hide which invariant fired.
func registryErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrRoomNotFound):
		return errRoomClosed
	case errors.Is(err, ErrOwnerHasRoom), errors.Is(err, ErrHandleInUse), errors.Is(err, ErrRoomFull):
		return errOtherRequests
	case errors.Is(err, ErrRoomFrozen):
		return errUnable
	default:
		return errUnable
	}
}

// CreateRoom opens a solo trade room for characterId (FR-1.1). The validation
// ladder runs in order — dead -> map -> level -> already-in-room — and each
// failure buffers the faithful mini-room enter error rather than returning, so
// the buffer still flushes and the client still hears why.
func (p *ProcessorImpl) CreateRoom(txId uuid.UUID, f field.Model, characterId character.Id, roomType byte) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.createRoom(mb, txId, f, characterId, roomType)
	})
}

func (p *ProcessorImpl) createRoom(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId character.Id, roomType byte) error {
	cm, err := p.cp.GetById(characterId)
	if err != nil {
		return err
	}
	if cm.Hp() == 0 {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, roomType, characterId, errNotWhenDead))
	}

	allowed, err := p.mapAllowsTrade(f.MapId())
	if err != nil {
		return err
	}
	if !allowed {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, roomType, characterId, errTradeNotAllowed))
	}

	cfg := p.cfg.Get(p.l, p.ctx)
	if int(cm.Level()) < cfg.MinTradeLevel() {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, roomType, characterId, errUnable))
	}

	if _, ok := p.reg.GetByMember(p.t, characterId); ok {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, roomType, characterId, errOtherRequests))
	}

	room := NewBuilder(roomType, characterId, cm.Name(), f).Build()
	if err = p.reg.Create(p.t, room); err != nil {
		p.l.WithError(err).Warnf("Unable to register a trade room for character [%d].", characterId)
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, roomType, characterId, registryErrorCode(err)))
	}

	recordRoomOpened(p.t)
	p.l.WithFields(p.roomFields(room)).Infof("Opened trade room [%s].", room.Id().String())
	return mb.Put(trademsg.EnvEventTopicStatus, roomCreatedProvider(txId, room))
}

// mapAllowsTrade reports whether a trade room may be opened on the map. An
// unreadable field limit is a REFUSAL, not a default-allow: a missing flag must
// never be read as "tradeable" (design §7). The read failure is logged at ERROR
// because it means atlas-data is unreachable, not that the map forbids trade.
func (p *ProcessorImpl) mapAllowsTrade(mapId _map.Id) (bool, error) {
	fieldLimit, err := p.mp.FieldLimit(mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read the field limit of map [%d]. Refusing the trade rather than assuming it is permitted.", mapId)
		return false, nil
	}
	return !mapdata.TradeDisallowed(fieldLimit), nil
}

// Invite offers the caller's room to targetCharacterId (FR-2.1). The caller must
// own a room that is still OPEN_SOLO or PENDING_INVITE — an invite can be
// re-issued after a decline, but never once the room is paired.
func (p *ProcessorImpl) Invite(txId uuid.UUID, f field.Model, characterId character.Id, targetCharacterId character.Id) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.invite(mb, txId, f, characterId, targetCharacterId)
	})
}

func (p *ProcessorImpl) invite(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId character.Id, targetCharacterId character.Id) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, 0, characterId, errUnable))
	}
	// Only the owner invites, and only while the room can still take a visitor.
	if room.OwnerId() != characterId || !invitableState(room.State()) {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errUnable))
	}
	if targetCharacterId == characterId {
		return mb.Put(trademsg.EnvEventTopicStatus, inviteRejectedProvider(txId, room, inviteResultCannotFind))
	}

	code, err := p.checkInviteTarget(room, targetCharacterId)
	if err != nil {
		return err
	}
	if code != "" {
		return mb.Put(trademsg.EnvEventTopicStatus, inviteRejectedProvider(txId, room, code))
	}

	// The transition is compare-and-set inside the write lock so a second INVITE
	// racing an ENTER_ROOM cannot move an already-paired room back to pending.
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if !invitableState(cur.State()) {
			return Room{}, ErrRoomFrozen
		}
		// The target is recorded as the room's admission ticket in the SAME
		// compare-and-set as the state move: ENTER_ROOM admits only this
		// character, and the wire handle alone must never be enough.
		return cur.WithState(StatePendingInvite).WithInvited(targetCharacterId), nil
	})
	if err != nil {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, registryErrorCode(err)))
	}

	inviterName := ""
	if pt, found := updated.ParticipantFor(characterId); found {
		inviterName = pt.Name()
	}
	if err = mb.Put(trademsg.EnvEventTopicStatus, inviteSentProvider(txId, updated, targetCharacterId, inviterName)); err != nil {
		return err
	}
	return mb.Put(invitemsg.EnvCommandTopic, inviteCommandProvider(txId, updated, targetCharacterId))
}

// invitableState reports whether a room in this state may still issue an invite.
func invitableState(s State) bool {
	return s == StateOpenSolo || s == StatePendingInvite
}

// checkInviteTarget runs the target-side half of the invite ladder. It returns
// the inviteResult KEY to refuse with, or "" when the target may be invited. A
// non-nil error is an infrastructure failure and aborts the command.
func (p *ProcessorImpl) checkInviteTarget(room Room, targetCharacterId character.Id) (string, error) {
	tm, err := p.cp.GetById(targetCharacterId)
	if err != nil {
		p.l.WithError(err).Infof("Unable to read invite target [%d]; refusing the trade invite.", targetCharacterId)
		return inviteResultCannotFind, nil
	}
	tf, err := p.locp.FieldOf(targetCharacterId)
	if err != nil {
		p.l.WithError(err).Infof("Unable to locate invite target [%d]; refusing the trade invite.", targetCharacterId)
		return inviteResultCannotFind, nil
	}
	// A target the inviter cannot see is reported as not found, which is what
	// the reference client's code 1 says.
	if !tf.Equals(room.Field()) {
		return inviteResultCannotFind, nil
	}
	if tm.Hp() == 0 {
		return inviteResultBusy, nil
	}
	if _, ok := p.reg.GetByMember(p.t, targetCharacterId); ok {
		return inviteResultBusy, nil
	}
	return "", nil
}

// DeclineInvite tears the originator's pending room down (FR-2.5, design §3.1):
// the reference client closes the inviter's dialog on a decline, so the room
// does not survive it.
//
// This is the CLIENT-originated decline — the COMMAND_TOPIC_TRADE
// DECLINE_INVITE arm — so it also retires the offer in atlas-invites, which has
// not heard about it (see inviteRejectCommandProvider for what a lingering
// invite costs). Use InviteRejected for the other direction.
func (p *ProcessorImpl) DeclineInvite(txId uuid.UUID, characterId character.Id, originatorId character.Id) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.declineInvite(mb, txId, characterId, originatorId, true)
	})
}

// InviteRejected tears the originator's pending room down after atlas-invites
// has ALREADY retired the invite — an explicit reject it processed, or FR-2.6's
// expiry, which its timeout task emits as a REJECTED status
// (services/atlas-invites/atlas.com/invites/invite/task.go:43-54).
//
// It deliberately does not send a reject command back: the invite is already
// gone, so atlas-invites would fail to locate it and log an error
// (invite/processor.go:220-229), and the reject would be pure round-trip.
func (p *ProcessorImpl) InviteRejected(txId uuid.UUID, characterId character.Id, originatorId character.Id) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.declineInvite(mb, txId, characterId, originatorId, false)
	})
}

// declineInvite is the shared body. retireInvite tells it whether atlas-invites
// still holds the offer and therefore needs a REJECT command.
func (p *ProcessorImpl) declineInvite(mb *message.Buffer, txId uuid.UUID, characterId character.Id, originatorId character.Id, retireInvite bool) error {
	room, ok := p.reg.GetByMember(p.t, originatorId)
	if !ok {
		p.l.Debugf("Character [%d] declined a trade invite from [%d], which no longer has a room.", characterId, originatorId)
		return nil
	}
	if room.OwnerId() != originatorId || room.State() != StatePendingInvite {
		p.l.Debugf("Character [%d] declined a trade invite from [%d], whose room is in state [%s]. Ignoring.", characterId, originatorId, room.State())
		return nil
	}

	if retireInvite {
		if err := mb.Put(invitemsg.EnvCommandTopic, inviteRejectCommandProvider(txId, room, characterId)); err != nil {
			return err
		}
	}
	// A PENDING_INVITE room is frozen against staging, so it should hold no
	// reservations — but teardownRoom releases unconditionally rather than
	// assuming, because a leaked hold is invisible until a player complains
	// that an item will not move.
	return p.teardownRoom(mb, txId, room, originatorId, ReasonTradeCancelled)
}

// EnterRoom seats characterId at position 1 of the room owning the given wire
// handle and moves it to OPEN (FR-1.5, FR-2.4). The dead / map / level ladder is
// re-run for the entering character: the invite may have been accepted seconds
// after it was sent, from a different situation than it was issued in (FR-4.5,
// FR-4.6, FR-4.7) — including from a different map, which is why the same-field
// check reads the enterer's live location rather than trusting f.
//
// f addresses the ROOM-IS-GONE error only: with no room there is no field to
// take one from, so the caller's is the only one available.
//
// roomType is the kind of room the CALLER believes it is entering — the client
// dialog's own room type on the CASH_TRADE_OPEN path, the room's own on the
// invite-accept path. A mismatch means the request is not for this room.
func (p *ProcessorImpl) EnterRoom(txId uuid.UUID, f field.Model, characterId character.Id, handle uint32, roomType byte) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.enterRoom(mb, txId, f, characterId, handle, roomType)
	})
}

func (p *ProcessorImpl) enterRoom(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId character.Id, handle uint32, roomType byte) error {
	room, ok := p.reg.GetByHandle(p.t, handle)
	if !ok {
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, 0, characterId, errRoomClosed))
	}
	// THE ADMISSION GATE. The handle is the owner's character id (design §2.3)
	// and therefore public, so possession of it proves nothing: only the
	// character the owner's outstanding INVITE named may be seated. Without
	// this, anyone in the map could send handle = victimCharacterId and take
	// the invitee's seat before the invite was answered.
	//
	// The refusal is ROOM_CLOSED — the same answer an unknown handle gets — so
	// probing cannot distinguish "no such room" from "not your room".
	if !room.Admits(characterId) {
		p.l.Warnf("Character [%d] tried to enter trade room [%d], which they were not invited to.", characterId, handle)
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, 0, characterId, errRoomClosed))
	}
	// The room the handle resolved to must be the kind the caller asked for: a
	// cash-trade enter must not seat anyone in a plain trade room.
	if room.RoomType() != roomType {
		p.l.Warnf("Character [%d] tried to enter trade room [%d] as roomType [%d], but it is roomType [%d].", characterId, handle, roomType, room.RoomType())
		return mb.Put(trademsg.EnvEventTopicStatus, errorProvider(txId, f, 0, characterId, errRoomClosed))
	}
	// Already seated somewhere — including in THIS room, which a duplicate
	// accept would produce.
	if _, ok = p.reg.GetByMember(p.t, characterId); ok {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errOtherRequests))
	}
	// A paired room is closed: only a room still awaiting its invitee admits a
	// second character.
	if room.State() != StatePendingInvite || room.VisitorId() != 0 {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errOtherRequests))
	}
	// Where the enterer ACTUALLY is, read from atlas-maps — not the f the caller
	// handed us. On the invite-accept path f is derived from the room itself, so
	// comparing it to the room would be a tautology and a character who changed
	// map between invite and accept would be seated regardless.
	ef, err := p.locp.FieldOf(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character [%d] entering trade room [%d]. Refusing rather than seating them blind.", characterId, handle)
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errUnable))
	}
	if !ef.Equals(room.Field()) {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errNotSameMap))
	}

	cm, err := p.cp.GetById(characterId)
	if err != nil {
		return err
	}
	if cm.Hp() == 0 {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errNotWhenDead))
	}

	allowed, err := p.mapAllowsTrade(room.Field().MapId())
	if err != nil {
		return err
	}
	if !allowed {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errTradeNotAllowed))
	}

	cfg := p.cfg.Get(p.l, p.ctx)
	if int(cm.Level()) < cfg.MinTradeLevel() {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, errUnable))
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if cur.State() != StatePendingInvite {
			return Room{}, ErrRoomFrozen
		}
		if cur.VisitorId() != 0 {
			return Room{}, ErrRoomFull
		}
		// Re-check the ticket under the write lock: the ladder above ran on a
		// snapshot, and a re-issued INVITE between then and here would have
		// moved the ticket to someone else.
		if !cur.Admits(characterId) {
			return Room{}, ErrRoomFrozen
		}
		return cur.WithVisitor(characterId, cm.Name()).WithState(StateOpen), nil
	})
	if err != nil {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, registryErrorCode(err)))
	}

	return mb.Put(trademsg.EnvEventTopicStatus, participantEnteredProvider(txId, updated, characterId, cm.Name()))
}

// Chat relays one line of trade-room chat to both sides (FR-8.1). The room is
// resolved from the SPEAKER's membership rather than from any id the client
// sent, so a character who is not in a trade room cannot address one — the
// serverbound CHAT mode fans out to every mini-room family in atlas-channel, so
// most of the traffic arriving here belongs to a mini-game or a shop and is
// silently dropped.
//
// Chat is legal in every non-terminal state, including while settling: it
// mutates nothing and the client's dialog is still open.
func (p *ProcessorImpl) Chat(txId uuid.UUID, characterId character.Id, text string) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.chat(mb, txId, characterId, text)
	})
}

func (p *ProcessorImpl) chat(mb *message.Buffer, txId uuid.UUID, characterId character.Id, text string) error {
	if text == "" {
		return nil
	}
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	speaker, ok := room.ParticipantFor(characterId)
	if !ok {
		return nil
	}
	return mb.Put(trademsg.EnvEventTopicStatus, chatProvider(txId, room, characterId, speaker.Position(), text))
}

// TeardownCharacter removes the room the character occupies and tells both sides
// why (design §3.3). A room already in SETTLING is left alone — cancel loses to
// settlement (FR-6.5), and the saga's terminal status produces that room's
// client-visible LEAVE instead.
//
// reason is a leaveReason KEY string; callers pass the one their trigger implies
// (Reason* in this package).
func (p *ProcessorImpl) TeardownCharacter(txId uuid.UUID, characterId character.Id, reason string) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.teardownCharacter(mb, txId, characterId, reason)
	})
}

func (p *ProcessorImpl) teardownCharacter(mb *message.Buffer, txId uuid.UUID, characterId character.Id, reason string) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if room.State() == StateSettling {
		p.l.Infof("Ignoring trade teardown [%s] for character [%d]: room [%s] is already settling.", reason, characterId, room.Id().String())
		return nil
	}

	return p.teardownRoom(mb, txId, room, characterId, reason)
}

// stagedRelease is one reservation to cancel, with the slot it must be aimed at
// already resolved.
type stagedRelease struct {
	reservationId uuid.UUID
	characterId   character.Id
	inventoryType inventory.Type
	sourceSlot    slot.Position
}

// resolveStagedReleases works out which reservation to cancel, and where, for
// every item staged in the room on BOTH sides. It performs the REST reads and
// buffers NOTHING — the caller emits only after it has atomically claimed the
// room (see Registry.RemoveIf).
//
// Every path that abandons a room, settled or not, must end in these cancels.
// Under the reserve-at-staging model a staged asset never left its owner's
// inventory — it is merely held — so a room that disappears without cancelling
// leaves the owner unable to move, merge, drop or sell that stack until the TTL
// lapses, which is five minutes by default and is REFRESHED for as long as the
// process believes the room is alive.
//
// The slot is RE-RESOLVED rather than taken from the staged item; see
// resolveStagedSlot for why a recorded slot cannot be trusted.
func (p *ProcessorImpl) resolveStagedReleases(room Room) []stagedRelease {
	cache := p.newCompartmentCache()
	out := make([]stagedRelease, 0)
	for _, pt := range room.Participants() {
		for _, i := range pt.Items() {
			out = append(out, stagedRelease{
				reservationId: i.ReservationId(),
				characterId:   pt.CharacterId(),
				inventoryType: i.InventoryType(),
				sourceSlot:    p.resolveStagedSlot(cache, pt.CharacterId(), i),
			})
		}
	}
	return out
}

// withLateStages appends a release for any item staged into the room AFTER the
// releases were resolved — the window between the compartment reads and the
// claim, which is open on a room that was still OPEN when the teardown began.
//
// Their slots are the recorded ones rather than re-resolved: re-reading would
// mean another REST round trip after the room is already gone, and the recorded
// slot is right unless that same item ALSO moved inside the window. A cancel
// aimed at the wrong slot is a no-op; emitting none guarantees the leak.
func withLateStages(releases []stagedRelease, claimed Room) []stagedRelease {
	known := make(map[uuid.UUID]struct{}, len(releases))
	for _, r := range releases {
		known[r.reservationId] = struct{}{}
	}
	for _, pt := range claimed.Participants() {
		for _, i := range pt.Items() {
			if _, ok := known[i.ReservationId()]; ok {
				continue
			}
			releases = append(releases, stagedRelease{
				reservationId: i.ReservationId(),
				characterId:   pt.CharacterId(),
				inventoryType: i.InventoryType(),
				sourceSlot:    i.SourceSlot(),
			})
		}
	}
	return releases
}

// emitStagedReleases buffers the cancels resolveStagedReleases worked out.
//
// The cancel is a fire-and-forget command: atlas-inventory treats an unknown
// reservation as a no-op, so cancelling one that already expired is harmless,
// and cancelling twice is harmless for the same reason.
func (p *ProcessorImpl) emitStagedReleases(mb *message.Buffer, releases []stagedRelease) error {
	for _, r := range releases {
		if err := p.resp.CancelReservation(mb)(r.reservationId, r.characterId, r.inventoryType, r.sourceSlot); err != nil {
			return err
		}
	}
	return nil
}

// resolveStagedSlot returns the slot a staged item's asset CURRENTLY occupies.
//
// A staged item's recorded sourceSlot goes stale under ordinary play. Nothing
// stops a player rearranging their inventory while the trade dialog is open:
// atlas-channel's move handler has no mini-room gate
// (services/atlas-channel/atlas.com/channel/socket/handler/character_inventory_move.go:17-59),
// and atlas-inventory permits a plain swap of a reserved slot, re-keying the
// hold to the destination
// (services/atlas-inventory/atlas.com/inventory/compartment/processor.go:490-507).
// Only MERGE is reservation-guarded (canMergeAssets rule 4, :592-593).
//
// The reservation registry keys holds by (characterId, inventoryType, SLOT), so
// a cancel aimed at the vacated slot silently no-ops while the real hold lives
// on at the new slot until its TTL — and a refresh aimed there would file a
// fresh 300s hold on whatever item was swapped in. Asset ID is the only stable
// handle, so the slot is re-derived from it.
//
// Reacting to atlas-inventory's asset MOVED event instead would NOT close this:
// a swap relays the destination asset through temporarySlot() (math.MinInt16),
// and UpdateSlot suppresses any transition touching it
// (services/atlas-inventory/atlas.com/inventory/asset/processor.go:250-252), so
// an item swapped INTO the staged slot moves the staged asset out without
// emitting anything that names it.
//
// A failed read or a vanished asset falls back to the recorded slot: cancelling
// the wrong slot is a no-op, whereas cancelling nothing guarantees the leak.
//
// cache memoizes the compartment reads for the whole command; see
// compartmentCache for why that matters.
func (p *ProcessorImpl) resolveStagedSlot(cache *compartmentCache, characterId character.Id, i StagedItem) slot.Position {
	c, err := cache.get(characterId, i.InventoryType())
	if err != nil {
		p.l.WithError(err).Warnf("Unable to re-resolve the slot of staged asset [%d] for character [%d]; falling back to the staged slot [%d].", i.AssetId(), characterId, i.SourceSlot())
		return i.SourceSlot()
	}
	a, ok := c.FindById(i.AssetId())
	if !ok {
		p.l.Warnf("Staged asset [%d] is no longer in character [%d]'s compartment [%d]; falling back to the staged slot [%d].", i.AssetId(), characterId, i.InventoryType(), i.SourceSlot())
		return i.SourceSlot()
	}
	if a.Slot() != i.SourceSlot() {
		p.l.Infof("Staged asset [%d] moved from slot [%d] to [%d] in character [%d]'s inventory since it was staged.", i.AssetId(), i.SourceSlot(), a.Slot(), characterId)
	}
	return a.Slot()
}

// compartmentKey identifies one character's compartment within a single command.
type compartmentKey struct {
	characterId   character.Id
	inventoryType inventory.Type
}

// compartmentCache memoizes compartment reads for the duration of ONE command.
//
// Without it, resolveStagedSlot refetched a whole compartment per staged item:
// a room at the default cap of 9 items per side is up to 18 GETs to fetch at
// most 2 distinct compartments, and the refresh runs every live room serially
// inside a single database.ExecuteTransaction — so a pooled DB connection would
// be held open across O(18N) synchronous HTTP round trips for N live rooms. That
// also widens the refresh-vs-teardown window the liveness re-check guards.
//
// A staged item's compartment cannot change while it is staged (the compartment
// is part of the staged item's identity, and only the SLOT within it can move),
// so one read per (character, compartment) per command is sufficient.
//
// A failed read is memoized too: the fallback is the same for every item in that
// compartment, so retrying per item would only multiply the latency of an
// already-unreachable atlas-inventory.
type compartmentCache struct {
	p    *ProcessorImpl
	seen map[compartmentKey]inventorydata.Model
	errs map[compartmentKey]error
}

func (p *ProcessorImpl) newCompartmentCache() *compartmentCache {
	return &compartmentCache{
		p:    p,
		seen: make(map[compartmentKey]inventorydata.Model),
		errs: make(map[compartmentKey]error),
	}
}

func (c *compartmentCache) get(characterId character.Id, inventoryType inventory.Type) (inventorydata.Model, error) {
	k := compartmentKey{characterId: characterId, inventoryType: inventoryType}
	if m, ok := c.seen[k]; ok {
		return m, nil
	}
	if err, ok := c.errs[k]; ok {
		return inventorydata.Model{}, err
	}
	m, err := c.p.invp.GetCompartment(characterId, inventoryType)
	if err != nil {
		c.errs[k] = err
		return inventorydata.Model{}, err
	}
	c.seen[k] = m
	return m, nil
}

// PutItem stages one item into the caller's side of the dialog (FR-3.1..FR-3.3,
// FR-4.1..FR-4.4). Every rejection below is a SILENT drop: the reference client
// has no put-item-time error arm, so the empty dialog slot is the feedback and
// the server-side log is the diagnostic (design §7).
func (p *ProcessorImpl) PutItem(txId uuid.UUID, characterId character.Id, inventoryType byte, sourceSlot slot.Position, quantity uint16, targetSlot byte) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.putItem(mb, txId, characterId, inventoryType, sourceSlot, quantity, targetSlot)
	})
}

func (p *ProcessorImpl) putItem(mb *message.Buffer, txId uuid.UUID, characterId character.Id, inventoryType byte, sourceSlot slot.Position, quantity uint16, targetSlot byte) error {
	room, pt, ok := p.stageableRoom(characterId, "PUT_ITEM")
	if !ok {
		return nil
	}

	cfg := p.cfg.Get(p.l, p.ctx)
	if !stageSlotAvailable(pt, targetSlot, cfg.MaxStagedItems()) {
		p.l.Debugf("Character [%d] staged into trade slot [%d], which is out of the 1..%d range or already occupied. Dropping.", characterId, targetSlot, cfg.MaxStagedItems())
		return nil
	}

	// Decoded here because the reads below need the typed compartment.
	// checkRestrictions decodes the same byte again and its failure branch is
	// therefore unreachable from this path; that redundancy is deliberate, so
	// checkRestrictions stays safe to call from a site that has NOT already
	// validated the byte, and stays testable as the boundary in its own right.
	it, ok := stageableInventoryType(inventoryType)
	if !ok {
		p.l.Infof("Character [%d] staged from compartment byte [%d], which is not a stageable inventory. Dropping.", characterId, inventoryType)
		return nil
	}

	a, err := p.invp.AssetInSlot(characterId, it, sourceSlot)
	if err != nil {
		p.l.WithError(err).Infof("Unable to read the asset character [%d] staged from compartment [%d] slot [%d]. Dropping.", characterId, it, sourceSlot)
		return nil
	}

	if err = checkRestrictions(assetView{Flags: a.Flag(), SourceSlot: a.Slot()}, p.itemData(it, a.TemplateId()), inventoryType); err != nil {
		p.l.WithError(err).Infof("Refusing to stage item [%d] for character [%d].", a.TemplateId(), characterId)
		return nil
	}

	staged, ok := p.stageableQuantity(characterId, pt, it, sourceSlot, a, quantity)
	if !ok {
		return nil
	}

	// The reservation handle is minted BEFORE the registry swap so the staged
	// item can carry it; the command that files it is buffered afterwards.
	//
	// The swap and the command are NOT atomic with each other, and deliberately
	// so. The registry is in-memory and is not rolled back by the enclosing
	// transaction (see emit's doc comment), so if the outbox insert fails the
	// item is staged with no hold behind it. That exposure is self-healing: the
	// refresh ticker re-issues REQUEST_RESERVE for every staged item it finds,
	// so the missing hold is filed on the next tick.
	//
	// Buffering the reserve BEFORE the swap would trade that for a strictly
	// worse failure: a stage that then lost the compare-and-set race would leave
	// a reserve command already in the batch — an orphaned hold that no staged
	// item references and therefore nothing ever cancels. A buffer has no
	// un-put, so the ordering below is the one that fails safe.
	reservationId := uuid.New()
	stagedItem := NewStagedItem(targetSlot, a.Id(), a.TemplateId(), staged, it, sourceSlot, reservationId)

	// Compare-and-set: the frozen / slot / cap checks are re-run inside the
	// write lock so a PUT_ITEM racing a CONFIRM or another PUT_ITEM cannot
	// overshoot the cap or double-book a dialog slot.
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if cur.Frozen() {
			return Room{}, ErrRoomFrozen
		}
		cp, found := cur.ParticipantFor(characterId)
		if !found {
			return Room{}, ErrRoomNotFound
		}
		if !stageSlotAvailable(cp, targetSlot, cfg.MaxStagedItems()) {
			return Room{}, ErrRoomFrozen
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithItem(stagedItem)
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Character [%d]'s stage into trade slot [%d] lost a race. Dropping.", characterId, targetSlot)
		return nil
	}

	if err = p.resp.RequestReserve(mb)(reservationId, characterId, it, sourceSlot, a.TemplateId(), staged, cfg.ReservationTtl()); err != nil {
		return err
	}
	return mb.Put(trademsg.EnvEventTopicStatus, itemStagedProvider(txId, updated, characterId, pt.Position(), stagedItem))
}

// itemData resolves the atlas-data view a restriction check needs. A lookup
// FAILURE is reported as Unreadable rather than as a tradeable default, and is
// logged at ERROR because it means atlas-data is unreachable — not that the item
// forbids trading (design §7).
func (p *ProcessorImpl) itemData(inventoryType inventory.Type, templateId item.Id) itemDataView {
	blocked, err := p.idp.TradeBlock(inventoryType, templateId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read the trade restrictions of item [%d]. Refusing the stage rather than assuming it is tradeable.", templateId)
		return itemDataView{Unreadable: true}
	}
	return itemDataView{TradeBlock: blocked}
}

// stageableRoom resolves the room a staging command may act on. A missing room
// is a DEBUG-level drop (the command simply arrived late), while a frozen room
// is a WARN: the reference client blocks staging locally once either side has
// confirmed, so reaching the server means a modified client (FR-3.6).
func (p *ProcessorImpl) stageableRoom(characterId character.Id, what string) (Room, Participant, bool) {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued %s without a trade room. Dropping.", characterId, what)
		return Room{}, Participant{}, false
	}
	if room.Frozen() {
		p.l.Warnf("Character [%d] issued %s against a frozen trade room [%s]; staging after confirm indicates a modified client. Dropping.", characterId, what, room.Id().String())
		return Room{}, Participant{}, false
	}
	pt, ok := room.ParticipantFor(characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued %s but is not seated in room [%s]. Dropping.", characterId, what, room.Id().String())
		return Room{}, Participant{}, false
	}
	return room, pt, true
}

// stageSlotAvailable reports whether targetSlot is a free dialog slot within the
// configured cap (FR-3.3, FR-9.1). The dialog's slots are 1-based, so 0 is out
// of range rather than "the first slot".
func stageSlotAvailable(pt Participant, targetSlot byte, maxStagedItems int) bool {
	if maxStagedItems <= 0 {
		return false
	}
	if targetSlot < 1 || int(targetSlot) > maxStagedItems {
		return false
	}
	if pt.HasTradeSlot(targetSlot) {
		return false
	}
	return len(pt.Items()) < maxStagedItems
}

// stageableQuantity validates the client's requested quantity against what the
// asset actually holds, net of what this room already claimed out of the same
// slot. It returns the quantity to stage.
func (p *ProcessorImpl) stageableQuantity(characterId character.Id, pt Participant, inventoryType inventory.Type, sourceSlot slot.Position, a inventorydata.Asset, quantity uint16) (asset.Quantity, bool) {
	// The reserve command's wire quantity is int16, narrower than the uint16
	// the client's PUT_ITEM decodes; anything above math.MaxInt16 would wrap
	// negative and be widened by atlas-inventory into a near-4-billion hold.
	if quantity == 0 || quantity > math.MaxInt16 {
		p.l.Infof("Character [%d] staged quantity [%d], outside the reservable 1..%d range. Dropping.", characterId, quantity, math.MaxInt16)
		return 0, false
	}
	want := asset.Quantity(quantity)
	claimed := pt.StagedQuantityFrom(inventoryType, sourceSlot)
	if claimed >= a.Quantity() || a.Quantity()-claimed < want {
		p.l.Infof("Character [%d] staged [%d] of the asset in compartment [%d] slot [%d], which holds [%d] with [%d] already claimed. Dropping.", characterId, want, inventoryType, sourceSlot, a.Quantity(), claimed)
		return 0, false
	}
	return want, true
}

// AddMeso assigns the caller's staged meso (FR-3.4, FR-4.8). Unlike PutItem, an
// out-of-range amount is NOT silent: mode 16 is an assignment on the client, so
// the client has already moved its own view and only an authoritative re-echo of
// the last valid amount will snap it back (design §4.2).
func (p *ProcessorImpl) AddMeso(txId uuid.UUID, characterId character.Id, amount int32) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.addMeso(mb, txId, characterId, amount)
	})
}

func (p *ProcessorImpl) addMeso(mb *message.Buffer, txId uuid.UUID, characterId character.Id, amount int32) error {
	room, pt, ok := p.stageableRoom(characterId, "ADD_MESO")
	if !ok {
		return nil
	}

	// A negative amount is not a mis-click, it is a forged packet — the client's
	// own input box cannot produce one. There is nothing to re-echo, because the
	// client never rendered a value we would be correcting.
	if amount < 0 {
		p.l.Warnf("Character [%d] staged a negative meso amount [%d]; a negative cannot come from the client's input box. Dropping.", characterId, amount)
		return nil
	}

	refused, err := p.mesoRefused(room, characterId, uint32(amount))
	if err != nil {
		return err
	}
	if refused {
		return mb.Put(trademsg.EnvEventTopicStatus, mesoRefusedProvider(txId, room, characterId, pt.Position(), pt.MesoStaged()))
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		if cur.Frozen() {
			return Room{}, ErrRoomFrozen
		}
		cp, found := cur.ParticipantFor(characterId)
		if !found {
			return Room{}, ErrRoomNotFound
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithMesoStaged(uint32(amount))
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Character [%d]'s meso stage lost a race. Dropping.", characterId)
		return nil
	}

	return mb.Put(trademsg.EnvEventTopicStatus, mesoStagedProvider(txId, updated, characterId, pt.Position(), uint32(amount)))
}

// mesoRefused reports whether the requested stage must be refused (FR-4.8). It
// reads the staging character's meso FRESH — a value cached at room-creation
// time would let a player stage meso they spent in the meantime — and, when a
// counterparty is seated, checks that the delivered amount would not push that
// counterparty over the meso ceiling.
//
// The counterparty check deliberately ignores the meso that side is itself
// staging outward: netting it in would depend on a value the counterparty can
// change afterwards, so this errs toward refusing, never toward accepting a
// trade that overflows on delivery.
func (p *ProcessorImpl) mesoRefused(room Room, characterId character.Id, amount uint32) (bool, error) {
	if amount > maxStageableMeso {
		p.l.Infof("Character [%d] staged [%d] meso, above the settleable ceiling of [%d]. Refusing.", characterId, amount, uint32(maxStageableMeso))
		return true, nil
	}

	cm, err := p.cp.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read character [%d]'s meso. Refusing the stage rather than trusting the client's amount.", characterId)
		return true, nil
	}
	if amount > cm.Meso() {
		p.l.Infof("Character [%d] staged [%d] meso but holds [%d]. Refusing.", characterId, amount, cm.Meso())
		return true, nil
	}

	other := counterpartyOf(room, characterId)
	if other == 0 {
		return false, nil
	}
	cfg := p.cfg.Get(p.l, p.ctx)
	_, delivered := configuration.Tax(cfg, amount)
	om, err := p.cp.GetById(other)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read counterparty [%d]'s meso. Refusing character [%d]'s stage rather than risking an overflow on delivery.", other, characterId)
		return true, nil
	}
	if uint64(om.Meso())+uint64(delivered) > maxStageableMeso {
		p.l.Infof("Character [%d] staged [%d] meso, which would deliver [%d] to counterparty [%d] who already holds [%d]. Refusing.", characterId, amount, delivered, other, om.Meso())
		return true, nil
	}
	return false, nil
}

// counterpartyOf returns the other seated participant, or 0 while the room is
// still solo.
func counterpartyOf(room Room, characterId character.Id) character.Id {
	for _, pt := range room.Participants() {
		if pt.CharacterId() != characterId {
			return pt.CharacterId()
		}
	}
	return 0
}

// RefreshReservations re-files every live room's reservations so none expires
// under an open trade (design §5.3).
//
// atlas-inventory has no refresh primitive, and its AddReservation APPENDS
// unconditionally
// (services/atlas-inventory/atlas.com/inventory/compartment/reservation_registry.go:110-125),
// so simply re-sending REQUEST_RESERVE would stack a second hold on the same
// slot every tick until the character's own stack read as fully reserved. The
// refresh is therefore CANCEL_RESERVATION followed by REQUEST_RESERVE under the
// same handle. Both commands are keyed by character id, so they share a
// partition and atlas-inventory processes them in that order.
func (p *ProcessorImpl) RefreshReservations() error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.refreshReservations(mb)
	})
}

func (p *ProcessorImpl) refreshReservations(mb *message.Buffer) error {
	ttl := p.cfg.Get(p.l, p.ctx).ReservationTtl()
	cache := p.newCompartmentCache()
	for _, snapshot := range p.reg.All(p.t) {
		// Cheap pre-filter only. The AUTHORITATIVE settling check is the one
		// inside correctStagedSlots' write-locked closure: this snapshot was
		// taken before the compartment reads below, so a room that settles
		// during them would pass here.
		if snapshot.State() == StateSettling {
			continue
		}

		// The staged slots are re-resolved by asset id (resolveStagedSlot) and
		// written back, so a player who rearranged their inventory mid-trade
		// does not leave the hold stranded at the vacated slot — and so the
		// slot the settlement payload is built from stays true.
		room, ok := p.correctStagedSlots(cache, snapshot)
		if !ok {
			continue
		}

		for _, pt := range room.Participants() {
			for _, i := range pt.Items() {
				if err := p.resp.CancelReservation(mb)(i.ReservationId(), pt.CharacterId(), i.InventoryType(), i.SourceSlot()); err != nil {
					return err
				}
				if err := p.resp.RequestReserve(mb)(i.ReservationId(), pt.CharacterId(), i.InventoryType(), i.SourceSlot(), i.TemplateId(), i.Quantity(), ttl); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// correctStagedSlots re-resolves every staged item's slot against its asset id
// and writes the corrections back, returning the room to refresh from.
//
// It doubles as the refresh's REFRESHABILITY re-check, covering both ways a
// snapshot can go stale while the compartment reads are in flight:
//
//   - The room was TORN DOWN. reg.All returns a snapshot and a concurrent
//     teardown emits its cancels from a different transaction, so refreshing
//     from the snapshot would re-file a 300s hold on a room that no longer
//     exists and nothing would ever cancel it. Registry.Update reports
//     ErrRoomNotFound once the room is gone.
//   - The room began SETTLING. Its holds are now the settlement saga's to
//     consume, and a CANCEL+REQUEST_RESERVE against them races that consume.
//     refreshReservations' own state test runs against the pre-read snapshot,
//     so this closure — which runs under the write lock, after the reads — is
//     the authoritative one.
//
// Doing both inside the closure makes the check and the correction one atomic
// step, rather than a check that could itself go stale before the emit.
//
// The REST reads happen BEFORE Update, never inside it: the closure runs under
// the registry write lock and must stay pure.
func (p *ProcessorImpl) correctStagedSlots(cache *compartmentCache, snapshot Room) (Room, bool) {
	corrections := make(map[uuid.UUID]slot.Position)
	for _, pt := range snapshot.Participants() {
		for _, i := range pt.Items() {
			if resolved := p.resolveStagedSlot(cache, pt.CharacterId(), i); resolved != i.SourceSlot() {
				corrections[i.ReservationId()] = resolved
			}
		}
	}

	room, err := p.reg.Update(p.t, snapshot.Id(), func(cur Room) (Room, error) {
		if cur.State() == StateSettling {
			return Room{}, ErrRoomFrozen
		}
		if len(corrections) == 0 {
			return cur, nil
		}
		return cur.WithEachParticipant(func(v Participant) Participant {
			return v.WithRelocatedItems(corrections)
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Skipping the reservation refresh of trade room [%s]: it is no longer refreshable.", snapshot.Id().String())
		return Room{}, false
	}
	return room, true
}

// RefreshReservations runs one refresh pass over the rooms of the tenant carried
// by ctx.
func RefreshReservations(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) error {
	return NewProcessor(l, ctx, db).RefreshReservations()
}

// minReservationRefreshInterval floors the refresh pace. A tenant is free to
// configure a very short — or, through a malformed resource, a zero —
// reservation TTL, and a pace derived from it unchecked would turn the ticker
// into a busy loop that republishes every hold continuously.
const minReservationRefreshInterval = time.Second

// ReservationRefreshInterval is how long may pass between two refresh passes for
// a tenant on the given configuration: a THIRD of its reservation TTL, so a hold
// survives two consecutively missed passes before it can lapse (design §5.3).
//
// It is a function of the configuration rather than a constant because the TTL
// is per-tenant (configuration.Model.ReservationTtl, default 300s).
func ReservationRefreshInterval(cfg configuration.Model) time.Duration {
	interval := cfg.ReservationTtl() / 3
	if interval < minReservationRefreshInterval {
		return minReservationRefreshInterval
	}
	return interval
}

// RefreshAllReservations runs one refresh pass for EVERY tenant that owns a live
// room, and returns how long the caller may wait before the next pass.
//
// It is the ticker's entry point, and it runs with no tenant in context: the
// registry is asked which tenants have rooms, and each pass runs under that
// tenant's own context so both its configuration and its registry partition
// resolve correctly.
//
// The interval comes back from the same pass that used it rather than from a
// separate enumeration, so the pace can never be derived from a different set of
// tenants than the one just refreshed. It is the SHORTEST interval any live
// tenant needs — a tenant with a 300s TTL being refreshed more often than it
// needs is free, whereas one with a 60s TTL paced by another tenant's 300s would
// lose its holds. With no live rooms it falls back to the shipped default, which
// is also the pace a first pass runs at.
//
// One tenant's failure does not stop the others: every tenant is attempted, and
// the error reports how many could not be refreshed. A refusal is not a failure
// — a room that settled or was torn down mid-pass is skipped inside
// RefreshReservations, by design.
func RefreshAllReservations(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) (time.Duration, error) {
	return refreshAllReservations(l, ctx, GetRegistry().Tenants(), configuration.GetRegistry(), func(tctx context.Context) error {
		return RefreshReservations(l, tctx, db)
	})
}

// refreshAllReservations is RefreshAllReservations' body with its three
// dependencies — the live tenants, the config source and the per-tenant pass —
// passed in, so the pacing and the keep-going-on-failure behaviour can be
// exercised without standing up atlas-tenants or a database.
//
// ctx is the process context the ticker runs under; each pass gets it with the
// tenant attached, so a shutdown cancels an in-flight pass rather than detaching
// it.
func refreshAllReservations(l logrus.FieldLogger, ctx context.Context, tenants []tenant.Model, cfgs configProvider, pass func(context.Context) error) (time.Duration, error) {
	next := ReservationRefreshInterval(configuration.DefaultConfig())

	var failures int
	for _, t := range tenants {
		tctx := tenant.WithContext(ctx, t)
		if interval := ReservationRefreshInterval(cfgs.Get(l, tctx)); interval < next {
			next = interval
		}
		if err := pass(tctx); err != nil {
			l.WithError(err).Errorf("Unable to refresh the trade reservations of tenant [%s]. Its holds lapse when their TTL does, which fails settlement with a clean LEAVE.", t.Id().String())
			failures++
		}
	}
	if failures > 0 {
		return next, fmt.Errorf("trade reservation refresh failed for %d tenant(s)", failures)
	}
	return next, nil
}

// reservationRefreshInitialDelay is how long the loop waits before its FIRST
// pass: nothing.
//
// The pace is REPORTED BY a pass, so before one has run there is no honest
// interval to wait — only a guess, and the shipped default (100s) is the WRONG
// guess for any tenant whose TTL is under 300s. A tenant configured at 60s that
// staged an item during that first window would have had its hold lapse at
// t≈65s while the first pass was still waiting for t=100s, which is exactly the
// failure the TTL/3 rule exists to prevent. Running immediately costs one
// registry read on an empty registry and replaces the guess with the real pace.
const reservationRefreshInitialDelay = time.Duration(0)

// RunReservationRefresh drives the reservation-refresh loop until ctx is
// cancelled, refreshing every live room's inventory holds so a trade window
// never outlives its reservations (design §5.3).
//
// It returns only on cancellation, so callers run it under routine.Go.
func RunReservationRefresh(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	runReservationRefresh(l, ctx, reservationRefreshInitialDelay, func(pctx context.Context) (time.Duration, error) {
		return RefreshAllReservations(l, pctx, db)
	})
}

// runReservationRefresh is RunReservationRefresh's body with the initial delay
// and the pass injected, so the pacing — including the first fire, which no
// pass has yet reported an interval for — can be exercised without a database.
//
// A timer rather than a ticker: each pass reports the interval the tenants it
// just refreshed actually need, and a ticker's period is fixed at construction.
func runReservationRefresh(l logrus.FieldLogger, ctx context.Context, initialDelay time.Duration, pass func(context.Context) (time.Duration, error)) {
	timer := time.NewTimer(initialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			next, err := pass(ctx)
			if err != nil {
				l.WithError(err).Error("Unable to refresh trade reservations.")
			}
			timer.Reset(next)
		}
	}
}
