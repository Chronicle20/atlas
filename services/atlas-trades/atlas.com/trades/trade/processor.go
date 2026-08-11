package trade

import (
	"atlas-trades/configuration"
	characterdata "atlas-trades/data/character"
	inventorydata "atlas-trades/data/inventory"
	itemdata "atlas-trades/data/item"
	"atlas-trades/data/location"
	mapdata "atlas-trades/data/map"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/escrow"
	"atlas-trades/kafka/message"
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"
	sagaproducer "atlas-trades/saga"
	"context"
	"errors"
	"fmt"
	"math"

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

// mesoActorType labels the award_mesos a meso stake produces. The player is
// moving their own meso into and out of escrow, so they are their own actor —
// the same labelling expandTradeSettlement uses for the delivery leg
// (saga/processor.go:1619). SYSTEM is reserved for the unwind refund, which no
// player asked for.
const mesoActorType = "CHARACTER"

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
// compartment when the settlement pre-checks need to know what the RECEIVING
// side can carry.
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

// sagaSubmitter buffers every saga command atlas-trades produces. atlas-trades
// submits COMPOSITES only; the orchestrator expands them (design §6.3, §5A.2).
type sagaSubmitter interface {
	Settle(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error
	Stage(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TransferToTradePayload) error
	StageMeso(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.AwardMesosPayload) error
	Unwind(mb *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeUnwindPayload) error
}

// escrowStore reads atlas-trades' own custody rows — the assets that have
// genuinely left their owners' compartments (design §5A.3).
//
// Under reserve-at-staging the staged asset was still in the owner's inventory,
// so settlement and teardown re-read it from atlas-inventory and had to cope
// with it having moved. Escrow makes the row itself authoritative and immutable
// once written, which is why this replaces the compartment re-resolution the old
// model needed rather than merely moving it.
type escrowStore interface {
	ItemById(escrowId uuid.UUID) (escrow.ItemModel, bool, error)
	ItemsByRoom(roomId uuid.UUID) ([]escrow.ItemModel, error)
	MesosByRoom(roomId uuid.UUID) ([]escrow.MesoModel, error)
	MesoByOwner(roomId uuid.UUID, ownerId character.Id) (int64, bool, error)

	// ClaimForReturn is the one WRITE on this seam, and it is here rather than on
	// a separate escrow processor because every caller that reads rows in order
	// to return them must claim them through the same handle it read them with.
	// It reports whether THIS caller won the exclusive right to submit the
	// trade_unwind for the row; see escrow.ClaimItemForReturn for why the
	// alternative — read, decide, then write — hands the owner the item twice.
	ClaimForReturn(escrowId uuid.UUID, txId uuid.UUID) (bool, error)

	// withTx rebinds the reads onto a transaction.
	//
	// This is not a convenience. Every escrow read in this package happens inside
	// emit's transaction, and a reader still bound to the ROOT handle would take a
	// SECOND pooled connection to answer — one the transaction is not holding and
	// cannot release. That is an outright deadlock whenever the pool is exhausted,
	// and it is deterministic at pool size 1. It also reads outside the
	// transaction, so a command could see escrow state its own earlier write in
	// the same command had already changed.
	withTx(tx *gorm.DB) escrowStore
}

// escrowReader binds the escrow package's tenant-scoped read closures to this
// processor's database handle and tenant.
type escrowReader struct {
	db *gorm.DB
	t  tenant.Model
}

func (e escrowReader) withTx(tx *gorm.DB) escrowStore {
	e.db = tx
	return e
}

// ItemById reports absence as (_, false, nil) rather than as an error: a
// transaction id that names no escrow row is the ordinary case on this path —
// it is how a settlement saga's status is told apart from a stage's.
func (e escrowReader) ItemById(escrowId uuid.UUID) (escrow.ItemModel, bool, error) {
	m, err := escrow.ItemById(e.db, e.t.Id())(escrowId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return escrow.ItemModel{}, false, nil
		}
		return escrow.ItemModel{}, false, err
	}
	return m, true, nil
}

func (e escrowReader) ItemsByRoom(roomId uuid.UUID) ([]escrow.ItemModel, error) {
	return escrow.ItemsByRoom(e.db, e.t.Id())(roomId)
}

func (e escrowReader) MesosByRoom(roomId uuid.UUID) ([]escrow.MesoModel, error) {
	return escrow.MesosByRoom(e.db, e.t.Id())(roomId)
}

func (e escrowReader) MesoByOwner(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
	return escrow.MesoByOwner(e.db, e.t.Id())(roomId, ownerId)
}

func (e escrowReader) ClaimForReturn(escrowId uuid.UUID, txId uuid.UUID) (bool, error) {
	return escrow.ClaimItemForReturn(e.db, e.t.Id())(escrowId, txId)
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

	// UnwindFailed and UnwindSucceeded are the trade_unwind's own id space. It
	// owns none of the other three, so without these its terminal status falls
	// through every probe and is swallowed — see UnwindFailed for what that
	// costs both custody columns.
	UnwindFailed(txId uuid.UUID, reason string) (bool, error)
	UnwindSucceeded(txId uuid.UUID) (bool, error)

	// ReconcileSettlements completes this tenant's settlements whose terminal
	// status was never seen — the trades a restart would otherwise lose.
	ReconcileSettlements() error

	// StageSucceeded confirms one staged item whose transfer_to_trade saga
	// completed: the escrow row now exists, so the item stops being pending and
	// is announced to both dialogs with ITEM_STAGED.
	//
	// Keyed by the ESCROW ROW ID, which is also the staging saga's transaction
	// id (see saga.BuildStage), so the terminal status needs no lookup table.
	StageSucceeded(txId uuid.UUID, escrowId uuid.UUID) (bool, error)

	// StageFailed frees the dialog slot a failed stage was holding and unlocks
	// the staging client with ITEM_REFUSED. Nothing was ever announced, so the
	// counterparty sees nothing at all.
	//
	// Returns false when escrowId names no pending stage, which is how the saga
	// status consumer tells a staging saga from a settlement one.
	StageFailed(txId uuid.UUID, escrowId uuid.UUID, reason string) (bool, error)

	// MesoStageSucceeded commits an in-flight meso stake whose award_mesos
	// completed, and announces it to both sides.
	MesoStageSucceeded(txId uuid.UUID, stakeId uuid.UUID) (bool, error)

	// MesoStageFailed abandons an in-flight meso stake, re-echoing the last
	// valid amount so the staking client's dialog snaps back.
	MesoStageFailed(txId uuid.UUID, stakeId uuid.UUID, reason string) (bool, error)

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
	esc    escrowStore
	sgp    sagaSubmitter
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
		esc:   escrowReader{db: db, t: tenant.MustFromContext(ctx)},
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
// transaction. The escrow reader is rebound too — see escrowStore.withTx for why
// leaving it on the root handle is a deadlock rather than an inefficiency.
func (p *ProcessorImpl) withTx(tx *gorm.DB) *ProcessorImpl {
	cp := *p
	cp.db = tx
	cp.esc = p.esc.withTx(tx)
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
		// CANNOT_FIND_CHARACTER's client arm reads no name, so none is carried.
		return mb.Put(trademsg.EnvEventTopicStatus, inviteRejectedProvider(txId, room, inviteResultCannotFind, ""))
	}

	code, targetName, err := p.checkInviteTarget(room, targetCharacterId)
	if err != nil {
		return err
	}
	if code != "" {
		return mb.Put(trademsg.EnvEventTopicStatus, inviteRejectedProvider(txId, room, code, targetName))
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
// the inviteResult KEY to refuse with — or "" when the target may be invited —
// and the target's name, which every refusal arm except CANNOT_FIND_CHARACTER's
// interpolates into its message. A non-nil error is an infrastructure failure
// and aborts the command.
func (p *ProcessorImpl) checkInviteTarget(room Room, targetCharacterId character.Id) (string, string, error) {
	tm, err := p.cp.GetById(targetCharacterId)
	if err != nil {
		// The name is unavailable precisely because the read failed; the arm
		// this refuses with reads no name, so nothing is lost.
		p.l.WithError(err).Infof("Unable to read invite target [%d]; refusing the trade invite.", targetCharacterId)
		return inviteResultCannotFind, "", nil
	}
	tf, err := p.locp.FieldOf(targetCharacterId)
	if err != nil {
		p.l.WithError(err).Infof("Unable to locate invite target [%d]; refusing the trade invite.", targetCharacterId)
		return inviteResultCannotFind, tm.Name(), nil
	}
	// A target the inviter cannot see is reported as not found, which is what
	// the reference client's code 1 says.
	if !tf.Equals(room.Field()) {
		return inviteResultCannotFind, tm.Name(), nil
	}
	if tm.Hp() == 0 {
		return inviteResultBusy, tm.Name(), nil
	}
	if _, ok := p.reg.GetByMember(p.t, targetCharacterId); ok {
		return inviteResultBusy, tm.Name(), nil
	}
	return "", tm.Name(), nil
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

// compartmentKey identifies one character's compartment within a single command.
type compartmentKey struct {
	characterId   character.Id
	inventoryType inventory.Type
}

// compartmentCache memoizes compartment reads for the duration of ONE command.
//
// It survived the move to escrow because the settlement pre-checks still read
// live compartments — not the STAGED items, which are escrowed and no longer in
// anyone's inventory, but the RECEIVING side's free slots and stackable merges
// (design §6.1 check 1). Those reads are per (character, compartment) and a
// settlement asks about both compartments of both sides, so without the memo one
// pre-check refetches the same compartment once per item it is checking, holding
// a pooled DB connection open across the round trips.
//
// A failed read is memoized too: the outcome is the same for every item in that
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

// refuseStage answers a stage that will not happen.
//
// EVERY refusal in putItem goes through here, and none of them may simply
// return nil. The stage stays player-visibly silent — the trade slot is left
// empty, which is the feedback §7 specifies — but the client armed
// CWvsContext::m_bExclRequestSent when it sent PUT_ITEM and
// CWvsContext::CanSendExclRequest refuses every later exclusive request,
// including ADD_MESO, until a packet clears it. A silent return therefore wedges
// the dialog for the rest of the session. That is the defect this whole task
// exists to fix, and it is reachable through each of these branches, not only
// through a failed saga (design §5A.6).
//
// It is addressed to the STAGING character alone: nothing was ever announced, so
// the counterparty has nothing to correct.
func (p *ProcessorImpl) refuseStage(mb *message.Buffer, txId uuid.UUID, room Room, characterId character.Id, position byte, targetSlot byte) error {
	if room.Id() == uuid.Nil {
		// No room to address the refusal to. See stageableRoom: the reference
		// client cannot reach this, and a modified one wedges only itself.
		return nil
	}
	return mb.Put(trademsg.EnvEventTopicStatus, itemRefusedProvider(txId, room, characterId, position, targetSlot))
}

func (p *ProcessorImpl) putItem(mb *message.Buffer, txId uuid.UUID, characterId character.Id, inventoryType byte, sourceSlot slot.Position, quantity uint16, targetSlot byte) error {
	room, pt, ok := p.stageableRoom(characterId, "PUT_ITEM")
	if !ok {
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	cfg := p.cfg.Get(p.l, p.ctx)
	if !stageSlotAvailable(pt, targetSlot, cfg.MaxStagedItems()) {
		p.l.Debugf("Character [%d] staged into trade slot [%d], which is out of the 1..%d range or already occupied. Dropping.", characterId, targetSlot, cfg.MaxStagedItems())
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	// Decoded here because the reads below need the typed compartment.
	// checkRestrictions decodes the same byte again and its failure branch is
	// therefore unreachable from this path; that redundancy is deliberate, so
	// checkRestrictions stays safe to call from a site that has NOT already
	// validated the byte, and stays testable as the boundary in its own right.
	it, ok := stageableInventoryType(inventoryType)
	if !ok {
		p.l.Infof("Character [%d] staged from compartment byte [%d], which is not a stageable inventory. Dropping.", characterId, inventoryType)
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	a, err := p.invp.AssetInSlot(characterId, it, sourceSlot)
	if err != nil {
		p.l.WithError(err).Infof("Unable to read the asset character [%d] staged from compartment [%d] slot [%d]. Dropping.", characterId, it, sourceSlot)
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	if err = checkRestrictions(assetView{Flags: a.Flag(), SourceSlot: a.Slot()}, p.itemData(it, a.TemplateId()), inventoryType); err != nil {
		p.l.WithError(err).Infof("Refusing to stage item [%d] for character [%d].", a.TemplateId(), characterId)
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	staged, ok := p.stageableQuantity(characterId, pt, it, sourceSlot, a, quantity)
	if !ok {
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	// The escrow row id is minted BEFORE the registry swap so the staged item
	// can carry it; the saga that fills the row is buffered afterwards.
	//
	// The swap and the command are NOT atomic with each other, and deliberately
	// so. The registry is in-memory and is not rolled back by the enclosing
	// transaction (see emit's doc comment), so if the outbox insert fails the
	// slot is held by a pending item whose saga was never submitted. That
	// exposure is bounded by the stage timeout, which reaps a pending item whose
	// terminal status never arrives.
	//
	// Buffering the saga BEFORE the swap would trade that for a strictly worse
	// failure: a stage that then lost the compare-and-set race would leave a
	// staging command already in the batch — an escrow row that no dialog slot
	// references and therefore nothing ever unwinds, with the player's item
	// inside it. A buffer has no un-put, so the ordering below is the one that
	// fails safe.
	escrowId := uuid.New()
	stagedItem := NewStagedItem(targetSlot, a.Id(), a.TemplateId(), staged, it, sourceSlot, escrowId)

	// Compare-and-set: the frozen / slot / cap checks are re-run inside the
	// write lock so a PUT_ITEM racing a CONFIRM or another PUT_ITEM cannot
	// overshoot the cap or double-book a dialog slot.
	_, err = p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
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
		return p.refuseStage(mb, txId, room, characterId, pt.Position(), targetSlot)
	}

	// NO ITEM_STAGED here. The item is pending: it holds its dialog slot against
	// a second PUT_ITEM but stays unannounced until the escrow row exists, so a
	// staging saga that fails never showed either dialog an item that was not
	// actually escrowed (design §5A.4). StageSucceeded does the announcing.
	return p.sgp.Stage(mb)(escrowId, sharedsaga.TransferToTradePayload{
		TransactionId:       escrowId,
		EscrowId:            escrowId,
		RoomId:              room.Id(),
		CharacterId:         uint32(characterId),
		TradeSlot:           targetSlot,
		SourceInventoryType: byte(it),
		SourceSlot:          int16(sourceSlot),
		AssetId:             uint32(a.Id()),
		Quantity:            uint32(staged),
	})
}

// StageSucceeded is the confirming half of putItem: the escrow row exists, so
// the item stops being pending and both dialogs finally hear about it.
func (p *ProcessorImpl) StageSucceeded(txId uuid.UUID, escrowId uuid.UUID) (bool, error) {
	handled := false
	err := p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		var err error
		handled, err = p.stageSucceeded(mb, txId, escrowId)
		return err
	})
	return handled, err
}

func (p *ProcessorImpl) stageSucceeded(mb *message.Buffer, txId uuid.UUID, escrowId uuid.UUID) (bool, error) {
	room, pt, i, ok := p.findStagedItem(escrowId)
	if !ok {
		// No dialog slot claims this row. Either it is not a stage at all — a
		// settlement's transaction id, which the caller distinguishes by this
		// false — or the room was torn down while the stage was in flight, in
		// which case the item is now escrowed with no trade left to deliver it
		// and nothing else will ever come looking for it.
		return p.returnOrphanedStage(mb, escrowId)
	}
	if !i.Pending() {
		// A redelivered terminal status. The announcement already went out;
		// repeating it would add a second copy of the item to both dialogs.
		p.l.Debugf("Stage [%s] is already confirmed. Ignoring the redelivery.", escrowId.String())
		return true, nil
	}

	// The escrow row is the only surviving copy of the asset — staging deleted
	// the compartment row before this status arrived — so both dialogs are drawn
	// from it. A confirmed stage whose row is missing cannot be rendered and must
	// not be announced: announcing a slot neither client can populate is exactly
	// the invisible-transfer window the trade dialog's consent depends on not
	// having.
	row, found, err := p.esc.ItemById(escrowId)
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("stage [%s] was confirmed but its escrow row is gone", escrowId.String())
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		cp, found := cur.ParticipantFor(pt.CharacterId())
		if !found {
			return Room{}, ErrRoomNotFound
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithConfirmedItem(escrowId)
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Unable to confirm stage [%s]; its room is gone. The teardown that removed it owns the unwind.", escrowId.String())
		return true, nil
	}
	return true, mb.Put(trademsg.EnvEventTopicStatus, itemStagedProvider(txId, updated, pt.CharacterId(), pt.Position(), i.Confirmed(), row.Snapshot()))
}

// StageFailed frees a dialog slot whose stage never made it into escrow.
func (p *ProcessorImpl) StageFailed(txId uuid.UUID, escrowId uuid.UUID, reason string) (bool, error) {
	handled := false
	err := p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		var err error
		handled, err = p.stageFailed(mb, txId, escrowId, reason)
		return err
	})
	return handled, err
}

func (p *ProcessorImpl) stageFailed(mb *message.Buffer, txId uuid.UUID, escrowId uuid.UUID, reason string) (bool, error) {
	room, pt, i, ok := p.findStagedItem(escrowId)
	if !ok {
		return false, nil
	}
	if !i.Pending() {
		// The saga failed AFTER the accept was acked, so the item is escrowed
		// and the orchestrator's reverse-walk owns returning it. Dropping the
		// dialog slot here would desynchronise the dialog from the escrow.
		p.l.Warnf("Stage [%s] failed after it was confirmed: [%s]. Leaving it to the saga's compensation.", escrowId.String(), reason)
		return true, nil
	}

	p.l.Infof("Stage [%s] for character [%d] failed: [%s]. Freeing trade slot [%d].", escrowId.String(), pt.CharacterId(), reason, i.TradeSlot())
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		cp, found := cur.ParticipantFor(pt.CharacterId())
		if !found {
			return Room{}, ErrRoomNotFound
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			return v.WithoutItem(escrowId)
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Stage [%s] failed and its room is already gone. Nothing to free.", escrowId.String())
		return true, nil
	}

	recordStageFailed(p.t, reason)
	// The refusal is the ONLY thing the staging client gets, and it has to
	// arrive: the client armed m_bExclRequestSent when it sent PUT_ITEM and
	// nothing else in this path will clear it (design §5A.6).
	return true, mb.Put(trademsg.EnvEventTopicStatus, itemRefusedProvider(txId, updated, pt.CharacterId(), pt.Position(), i.TradeSlot()))
}

// returnOrphanedStage hands back an item that reached escrow after its room was
// already gone.
//
// This is the item twin of refundOrphanedStake, and it exists because a teardown
// unwinds what is escrowed AT THAT MOMENT: a stage whose accept_to_trade has not
// yet written its row leaves the teardown nothing to find, and the row appears
// afterwards with nothing referencing it. Without this the item is stranded in
// the custody table permanently — the player's item, gone, with no error
// anywhere.
//
// It reports false when the row does not exist, which is what tells the caller
// this transaction id was never a stage. A row that exists but was already
// CLAIMED by the teardown's own unwind reports TRUE with nothing submitted: this
// was a stage — so the caller must not go on treating the status as a
// settlement's — but its return is already in flight.
func (p *ProcessorImpl) returnOrphanedStage(mb *message.Buffer, escrowId uuid.UUID) (bool, error) {
	row, found, err := p.esc.ItemById(escrowId)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	// The claim, not the read, is what makes this path exclusive. The teardown
	// that removed the room reads ItemsByRoom with no pending filter, so it sees
	// this row from the moment the custody consumer writes it — well before the
	// stage's terminal status gets here — and both paths would otherwise submit
	// a trade_unwind for it (see escrow.ClaimItemForReturn).
	unwindTxId := uuid.New()
	won, err := p.esc.ClaimForReturn(escrowId, unwindTxId)
	if err != nil {
		return false, err
	}
	if !won {
		p.l.Debugf("Orphaned trade escrow row [%s] is already being returned by another path. Ignoring this one.", escrowId.String())
		return true, nil
	}

	p.l.Warnf("Trade escrow row [%s] was accepted after its room [%s] ended. Returning item [%d] to character [%d].", escrowId.String(), row.RoomId().String(), row.TemplateId(), row.OwnerId())

	// No field lookup: an item return accepts to the owner's compartment, which
	// atlas-inventory addresses by character id alone. Only the meso legs of an
	// unwind need a world and channel, and this one has none.
	txId := uuid.New()
	return true, p.sgp.Unwind(mb)(txId, sharedsaga.TradeUnwindPayload{
		TransactionId: txId,
		Items: []sharedsaga.TradeUnwindItem{{
			OwnerId: row.OwnerId(),
			Item:    escrowItemPayload(row),
		}},
	})
}

// findStagedItem locates one staged item across the tenant's live rooms by its
// escrow row. Rooms are few and in memory, so the scan is cheaper than the
// index it would otherwise take to avoid it.
func (p *ProcessorImpl) findStagedItem(escrowId uuid.UUID) (Room, Participant, StagedItem, bool) {
	for _, room := range p.reg.All(p.t) {
		for _, pt := range room.OrderedParticipants() {
			if i, ok := pt.ItemByEscrow(escrowId); ok {
				return room, pt, i, true
			}
		}
	}
	return Room{}, Participant{}, StagedItem{}, false
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
//
// A REFUSAL STILL RETURNS THE ROOM when one was found, and the participant when
// the caller is seated in it. That is not a convenience — it is what lets the
// caller answer. A refused stage is player-visibly silent by design (§7), but
// the client armed m_bExclRequestSent on send and stays locked until a packet
// clears it, so a refusal that returns nothing addressable wedges the dialog for
// the rest of the session (§5A.6). Callers MUST check the bool, not the zero
// value of the Room.
//
// The one genuinely unanswerable case is the first: a character in no room has
// no room-scoped event to carry the unlock, and no field to address one to. It
// is also the only case the reference client cannot produce — it will not send a
// staging opcode with no dialog open — so the exposure is a modified client
// wedging itself.
func (p *ProcessorImpl) stageableRoom(characterId character.Id, what string) (Room, Participant, bool) {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		p.l.Debugf("Character [%d] issued %s without a trade room. Dropping.", characterId, what)
		return Room{}, Participant{}, false
	}
	pt, seated := room.ParticipantFor(characterId)
	if room.Frozen() {
		p.l.Warnf("Character [%d] issued %s against a frozen trade room [%s]; staging after confirm indicates a modified client. Dropping.", characterId, what, room.Id().String())
		return room, pt, false
	}
	if !seated {
		p.l.Debugf("Character [%d] issued %s but is not seated in room [%s]. Dropping.", characterId, what, room.Id().String())
		return room, Participant{}, false
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

// refuseMeso answers a meso stage that will not happen, with the authoritative
// re-echo of the last valid amount.
//
// Same obligation as refuseStage, reached through the other opcode: PutMoney
// arms m_bExclRequestSent too, so a silent return locks the dialog. Here the
// re-echo is doing double duty — it snaps the client's view back (mode 16
// assigns, so the client already moved) AND carries the unlock.
func (p *ProcessorImpl) refuseMeso(mb *message.Buffer, txId uuid.UUID, room Room, characterId character.Id, pt Participant) error {
	if room.Id() == uuid.Nil {
		return nil
	}
	return mb.Put(trademsg.EnvEventTopicStatus, mesoRefusedProvider(txId, room, characterId, pt.Position(), pt.MesoStaged()))
}

func (p *ProcessorImpl) addMeso(mb *message.Buffer, txId uuid.UUID, characterId character.Id, amount int32) error {
	room, pt, ok := p.stageableRoom(characterId, "ADD_MESO")
	if !ok {
		return p.refuseMeso(mb, txId, room, characterId, pt)
	}

	// A negative amount is a forged packet — the client's own input box cannot
	// produce one — so there is no rendered value to correct. The re-echo goes
	// out anyway, carrying the last VALID amount: it is the only thing that
	// clears the lock PutMoney armed on send (design §5A.6), and re-stating the
	// truth to a client that lied costs nothing.
	if amount < 0 {
		p.l.Warnf("Character [%d] staged a negative meso amount [%d]; a negative cannot come from the client's input box. Dropping.", characterId, amount)
		return p.refuseMeso(mb, txId, room, characterId, pt)
	}

	// A stake already in flight is not refused, it is superseded: the client can
	// retype the box faster than a saga round trip, and refusing the second
	// entry would re-echo a stale number over the one the player just typed.
	// The pending txId is what makes the older saga's terminal status a no-op
	// when it lands (see WithSettledMeso).
	refused, err := p.mesoRefused(room, characterId, uint32(amount))
	if err != nil {
		return err
	}
	if refused {
		return mb.Put(trademsg.EnvEventTopicStatus, mesoRefusedProvider(txId, room, characterId, pt.Position(), pt.MesoStaged()))
	}

	// Mode 16 assigns rather than accumulates (design §1.6), so the movement is
	// the DELTA against what this participant has already moved toward the
	// trade — and it is the ESCROW ROW, not the room, that is authoritative:
	// the room's mesoStaged only advances once a stake settles, so reading it
	// here would re-debit a stake still in flight (design §5A.5).
	//
	// "Already moved" is committed PLUS every delta still in flight, which is
	// the item column's StagedQuantityFrom rule applied to a balance. Netting
	// against the committed figure alone was the destruction bug: a player who
	// retypes the box before the first saga resolves gets the second delta
	// computed as if the first had never happened, both sagas debit in full,
	// and the difference is destroyed.
	// Read through the SAME escrow processor that arms the stake below, not the
	// fake-able escrowStore seam. Arming and netting have to see one store: a
	// test double that answered this read while the arms went elsewhere would
	// report nothing in flight and silently restore the very bug this nets out.
	staked, err := escrow.NewProcessor(p.l, p.ctx, p.db).EffectiveMesoByOwner(room.Id(), characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to read character [%d]'s staked meso. Refusing the stage rather than risking a double debit.", characterId)
		return mb.Put(trademsg.EnvEventTopicStatus, mesoRefusedProvider(txId, room, characterId, pt.Position(), pt.MesoStaged()))
	}

	delta := int64(amount) - staked
	if delta == 0 {
		// Nothing moves, but the client still armed its lock on send, so the
		// re-echo is not optional (design §5A.6). MESO_STAGED carries it.
		return mb.Put(trademsg.EnvEventTopicStatus, mesoStagedProvider(txId, room, characterId, pt.Position(), uint32(amount)))
	}

	// The stake is armed on the ESCROW ROW before the room hears about it, and
	// that ordering is the whole reason the row carries a pending state at all.
	//
	// The room is in memory. If it is torn down while this saga is in flight, the
	// only record of an amount the player has already been debited would go with
	// it — the unwind refunds the committed escrow, which does not yet include
	// this stake, and the meso would be silently kept by the house. The durable
	// arm means the terminal status can still resolve it with no room at all
	// (see resolveMesoStake's room-gone branch).
	//
	// The DELTA is armed alongside the stake, not re-derived when it resolves: an
	// ordinary teardown zeroes the row's committed Amount while leaving the stake
	// armed, so by resolution time the figure the delta was computed against is
	// gone.
	stakeId := uuid.New()
	if err = escrow.NewProcessor(p.l, p.ctx, p.db).ArmMesoStake(room.Id(), characterId, stakeId, uint32(amount), int32(delta)); err != nil {
		p.l.WithError(err).Errorf("Unable to arm character [%d]'s meso stake. Refusing rather than debiting meso nothing would record.", characterId)
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
			return v.WithPendingMeso(stakeId, uint32(amount))
		}), nil
	})
	if err != nil {
		p.l.WithError(err).Debugf("Character [%d]'s meso stage lost a race. Dropping.", characterId)
		return p.refuseMeso(mb, txId, room, characterId, pt)
	}

	// Negative debits the staking player, positive refunds them. ShowEffect is
	// false: the meso chat line belongs to loot and shop trades, not to moving
	// your own stake in and out of a dialog you are looking at.
	return p.sgp.StageMeso(mb)(stakeId, sharedsaga.AwardMesosPayload{
		CharacterId: uint32(characterId),
		WorldId:     updated.Field().WorldId(),
		ChannelId:   updated.Field().ChannelId(),
		ActorId:     uint32(characterId),
		ActorType:   mesoActorType,
		Amount:      int32(-delta),
		ShowEffect:  false,
	})
}

// MesoStageSucceeded commits an in-flight stake and announces it to both sides.
func (p *ProcessorImpl) MesoStageSucceeded(txId uuid.UUID, stakeId uuid.UUID) (bool, error) {
	handled := false
	err := p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		var err error
		handled, err = p.resolveMesoStake(mb, txId, stakeId, true, "")
		return err
	})
	return handled, err
}

// MesoStageFailed abandons an in-flight stake and snaps the staking client back.
func (p *ProcessorImpl) MesoStageFailed(txId uuid.UUID, stakeId uuid.UUID, reason string) (bool, error) {
	handled := false
	err := p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		var err error
		handled, err = p.resolveMesoStake(mb, txId, stakeId, false, reason)
		return err
	})
	return handled, err
}

func (p *ProcessorImpl) resolveMesoStake(mb *message.Buffer, txId uuid.UUID, stakeId uuid.UUID, settled bool, reason string) (bool, error) {
	ep := escrow.NewProcessor(p.l, p.ctx, p.db)
	row, found, err := ep.MesoStakeById(stakeId)
	if err != nil {
		return false, err
	}
	if !found {
		// Not a meso stake at all — this is how the saga status consumer tells a
		// stake apart from an item stage or a settlement.
		return false, nil
	}
	amount := row.Amount()

	// The DURABLE row decides first, and its claim is what makes a redelivered
	// terminal status inert. Only then does the room follow; if there is no room
	// left, the escrow is still consistent.
	var matched bool
	if settled {
		matched, err = ep.CommitMesoStake(row.RoomId(), row.OwnerId(), stakeId)
	} else {
		matched, err = ep.AbandonMesoStake(row.RoomId(), row.OwnerId(), stakeId)
	}
	if err != nil {
		return true, err
	}
	if !matched {
		p.l.Debugf("Meso stake [%s] was already resolved. Ignoring the redelivery.", stakeId.String())
		return true, nil
	}

	room, ok := p.reg.Get(p.t, row.RoomId())
	if !ok {
		// The room was torn down while the debit was in flight. Its unwind
		// refunded the COMMITTED escrow, which did not include this stake — so if
		// the debit landed, this is the only chance to give it back.
		if !settled {
			// Abandoning moved nothing into Amount, so there is nothing extra to
			// hand back and nothing extra to discharge.
			return true, p.discardOrphanedMeso(ep, row, 0)
		}
		p.l.Warnf("Meso stake [%s] of [%d] settled into a trade room [%s] that is already gone. Refunding character [%d].", stakeId.String(), amount, row.RoomId().String(), row.OwnerId())
		if err = p.refundOrphanedStake(mb, row); err != nil {
			return true, err
		}
		// The commit above added this stake's delta to a row whose custody is
		// over, and the refund has just handed that delta back — so the row must
		// be discharged of exactly it.
		return true, p.discardOrphanedMeso(ep, row, row.Delta())
	}

	pt, found := room.ParticipantFor(row.OwnerId())
	if !found {
		return true, nil
	}

	resolved := false
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		cp, ok := cur.ParticipantFor(row.OwnerId())
		if !ok {
			return Room{}, ErrRoomNotFound
		}
		return cur.WithParticipant(cp.Position(), func(v Participant) Participant {
			next, ok := v.WithSettledMeso(stakeId, settled)
			resolved = ok
			return next
		}), nil
	})
	if err != nil || !resolved {
		// The room went, or a newer stake superseded this one between the read
		// and the write lock. The durable row is already correct either way.
		return true, nil
	}

	if !settled {
		p.l.Infof("Meso stake [%s] for character [%d] failed: [%s]. Re-echoing [%d].", stakeId.String(), row.OwnerId(), reason, pt.MesoStaged())
		return true, mb.Put(trademsg.EnvEventTopicStatus, mesoRefusedProvider(txId, updated, row.OwnerId(), pt.Position(), pt.MesoStaged()))
	}
	return true, mb.Put(trademsg.EnvEventTopicStatus, mesoStagedProvider(txId, updated, row.OwnerId(), pt.Position(), amount))
}

// refundOrphanedStake hands back a stake whose debit landed after its room was
// already torn down, as a one-meso trade_unwind.
//
// What has to be refunded is the DELTA the saga actually moved, not the absolute
// stake: the committed part was already returned by the teardown's own unwind,
// and refunding the total on top of it would mint meso.
//
// That delta is read off the stake, where ArmMesoStake recorded it, rather than
// re-derived from the row's committed Amount. Re-deriving it was the bug: the
// teardown that orphaned this stake also ZEROED Amount (clearRefundedMesos), so
// the subtraction meant to net out the already-refunded part netted out nothing
// and the whole stake went back on top of it.
func (p *ProcessorImpl) refundOrphanedStake(mb *message.Buffer, row escrow.MesoStakeModel) error {
	delta := int64(row.Delta())
	if delta <= 0 {
		// The stake was a REDUCTION, so its saga credited the player already.
		// Nothing is owed.
		return nil
	}
	// The owner's CURRENT field, not the dead room's. The world and channel steer
	// where the meso update is rendered, and a teardown is very often a channel
	// change — announcing onto the room's old channel would credit the meso where
	// the player no longer is.
	f, err := p.locp.FieldOf(row.OwnerId())
	if err != nil {
		return fmt.Errorf("unable to locate character %d to refund an orphaned meso stake of %d: %w", row.OwnerId(), delta, err)
	}
	txId := uuid.New()
	return p.sgp.Unwind(mb)(txId, sharedsaga.TradeUnwindPayload{
		TransactionId: txId,
		Mesos: []sharedsaga.TradeUnwindMeso{{
			CharacterId: row.OwnerId(),
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      uint32(delta),
		}},
	})
}

// discardOrphanedMeso discharges the escrow row an ORPHANED stake just resolved
// against — the row whose room is already gone — of the amount that stake put
// there, and retires the row once nothing it records is outstanding.
//
// It is what stops a second refund. The commit added this stake's delta to
// Amount, which is right while a room still holds the custody that figure
// describes and stale the moment none does: the teardown that orphaned the
// stake already returned the committed part and refundOrphanedStake has just
// returned the delta, so what the commit left behind is escrow nobody holds.
// The next boot's ReconcileEscrow would read exactly that figure as a stranded
// asset and hand it back AGAIN.
//
// refunded is the amount to discharge — this stake's delta on the settled path,
// and zero on the abandoned path, where nothing was added in the first place.
//
// The subtraction is RELATIVE and applied by the database, never an assignment
// of a total computed here. That is the whole correction over the version this
// replaces, which decided from a committed figure read BEFORE the
// compare-and-set and then assigned: under READ COMMITTED — the isolation this
// fleet actually runs at, verified against the live cluster — a sibling stake
// committing or a teardown zeroing in that window made the decision wrong, and
// the assignment then clobbered whichever write had landed in between. Relative
// arithmetic composes with both.
//
// Discharged and then conditionally deleted rather than deleted outright,
// because the two are not the same guarantee. The delete refuses to touch a row
// with any stake still outstanding (DeleteResolvedMeso), so a row whose sibling
// stake is still in flight survives — and for that row the discharge is still
// what keeps a stale total out of the next sweep.
func (p *ProcessorImpl) discardOrphanedMeso(ep escrow.Processor, row escrow.MesoStakeModel, refunded int32) error {
	if refunded != 0 {
		if err := ep.DischargeMeso(row.RoomId(), row.OwnerId(), refunded); err != nil {
			return err
		}
	}
	_, err := ep.DeleteResolvedMeso(row.RoomId(), row.OwnerId())
	return err
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
