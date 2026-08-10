package trade

import (
	"atlas-trades/configuration"
	characterdata "atlas-trades/data/character"
	"atlas-trades/data/location"
	mapdata "atlas-trades/data/map"
	"atlas-trades/kafka/message"
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
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
const (
	ReasonTradeCancelled    = "TRADE_CANCELLED"     // 2
	ReasonTradeSuccess      = "TRADE_SUCCESS"       // 7
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

	// DeclineInvite tears down the originator's pending room (FR-2.5).
	DeclineInvite(txId uuid.UUID, characterId character.Id, originatorId character.Id) error

	// EnterRoom seats characterId in the room owning the given wire handle
	// (FR-1.5, FR-2.4).
	EnterRoom(txId uuid.UUID, f field.Model, characterId character.Id, handle uint32) error

	// TeardownCharacter removes the character's room, unless it is already
	// settling (design §3.3, "cancel loses to settlement").
	TeardownCharacter(txId uuid.UUID, characterId character.Id, reason string) error
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	reg  *Registry
	cfg  configProvider
	cp   characterProvider
	mp   mapProvider
	locp locationProvider
}

// NewProcessor resolves the tenant from ctx once; every registry read the
// processor issues is partitioned by that tenant.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		reg:  GetRegistry(),
		cfg:  configuration.GetRegistry(),
		cp:   characterdata.NewProcessor(l, ctx),
		mp:   mapdata.NewProcessor(l, ctx),
		locp: location.NewProcessor(l, ctx),
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
		return cur.WithState(StatePendingInvite), nil
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
// does not survive it. It also covers FR-2.6 — atlas-invites emits REJECTED for
// a timed-out invite (services/atlas-invites/atlas.com/invites/invite/task.go:51),
// so expiry lands here too.
func (p *ProcessorImpl) DeclineInvite(txId uuid.UUID, characterId character.Id, originatorId character.Id) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.declineInvite(mb, txId, characterId, originatorId)
	})
}

func (p *ProcessorImpl) declineInvite(mb *message.Buffer, txId uuid.UUID, characterId character.Id, originatorId character.Id) error {
	room, ok := p.reg.GetByMember(p.t, originatorId)
	if !ok {
		p.l.Debugf("Character [%d] declined a trade invite from [%d], which no longer has a room.", characterId, originatorId)
		return nil
	}
	if room.OwnerId() != originatorId || room.State() != StatePendingInvite {
		p.l.Debugf("Character [%d] declined a trade invite from [%d], whose room is in state [%s]. Ignoring.", characterId, originatorId, room.State())
		return nil
	}

	p.reg.Remove(p.t, room.Id())
	return mb.Put(trademsg.EnvEventTopicStatus, cancelledProvider(txId, room, originatorId, ReasonTradeCancelled))
}

// EnterRoom seats characterId at position 1 of the room owning the given wire
// handle and moves it to OPEN (FR-1.5, FR-2.4). The dead / map / level ladder is
// re-run for the entering character: the invite may have been accepted seconds
// after it was sent, from a different situation than it was issued in (FR-4.5,
// FR-4.6, FR-4.7).
func (p *ProcessorImpl) EnterRoom(txId uuid.UUID, f field.Model, characterId character.Id, handle uint32) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.enterRoom(mb, txId, f, characterId, handle)
	})
}

func (p *ProcessorImpl) enterRoom(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId character.Id, handle uint32) error {
	room, ok := p.reg.GetByHandle(p.t, handle)
	if !ok {
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
	if !f.Equals(room.Field()) {
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
		return cur.WithVisitor(characterId, cm.Name()).WithState(StateOpen), nil
	})
	if err != nil {
		return mb.Put(trademsg.EnvEventTopicStatus, roomErrorProvider(txId, room, characterId, registryErrorCode(err)))
	}

	return mb.Put(trademsg.EnvEventTopicStatus, participantEnteredProvider(txId, updated, characterId, cm.Name()))
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

	p.reg.Remove(p.t, room.Id())
	return mb.Put(trademsg.EnvEventTopicStatus, cancelledProvider(txId, room, characterId, reason))
}
