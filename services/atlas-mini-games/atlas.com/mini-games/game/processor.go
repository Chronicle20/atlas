package game

import (
	chalkboarddata "atlas-mini-games/data/chalkboard"
	characterdata "atlas-mini-games/data/character"
	inventorydata "atlas-mini-games/data/inventory"
	mapdata "atlas-mini-games/data/map"
	"atlas-mini-games/kafka/message"
	"atlas-mini-games/kafka/message/minigame"
	kproducer "atlas-mini-games/kafka/producer"
	"atlas-mini-games/record"
	"context"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Room type discriminators carried by the CREATE command and every room event.
const (
	RoomTypeOmok       byte = 1
	RoomTypeMatchCards byte = 2
)

// Creation item ids (NOT consumed): Omok pieces are 4080000+pieceType (piece
// [0,11]); the single Match Cards set is 4080100 (spec [0,2]).
const (
	omokItemBase      uint32 = 4080000
	matchCardsItemId  uint32 = 4080100
	omokPieceMax      byte   = 11
	matchCardsSpecMax byte   = 2
)

// enterError key strings (channel resolves them to numeric codes via the tenant
// enterError table).
const (
	errNotWhenDead            = "NOT_WHEN_DEAD"
	errCannotStartGameHere    = "CANNOT_START_GAME_HERE"
	errCannotOpenMiniRoomHere = "CANNOT_OPEN_MINI_ROOM_HERE"
	errUnable                 = "UNABLE"
	errRoomClosed             = "ROOM_CLOSED"
	errFull                   = "FULL"
	errIncorrectPassword      = "INCORRECT_PASSWORD"
)

// Leave statuses (LeftEventBody.Status / RoomClosedEventBody.VisitorStatus).
const (
	leaveStatusClosed   byte = 3
	leaveStatusLeft     byte = 4
	leaveStatusExpelled byte = 5
)

// characterProvider, mapProvider, inventoryProvider and chalkboardProvider are
// the small REST-client seams the processor validates through, injected so
// tests can fake them.
type characterProvider interface {
	Hp(characterId uint32) (uint16, error)
}

type mapProvider interface {
	FieldLimit(mapId _map.Id) (uint32, error)
}

type inventoryProvider interface {
	HasItem(characterId uint32, itemId uint32) (bool, error)
}

type chalkboardProvider interface {
	HasOpen(characterId uint32) (bool, error)
}

// Processor handles mini-game lifecycle commands. Gameplay commands
// (READY/START/MOVE_STONE/FLIP_CARD/tie/retreat/SKIP/EXIT_AFTER_GAME) are added
// in Task 15.
type Processor interface {
	Create(txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error
	Visit(txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error
	Leave(txId uuid.UUID, f field.Model, characterId uint32) error
	Chat(txId uuid.UUID, f field.Model, characterId uint32, message string) error
	Expel(txId uuid.UUID, f field.Model, characterId uint32) error
	TeardownCharacter(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	reg *Registry
	cp  characterProvider
	mp  mapProvider
	ip  inventoryProvider
	chp chalkboardProvider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		reg: GetRegistry(),
		cp:  characterdata.NewProcessor(l, ctx),
		mp:  mapdata.NewProcessor(l, ctx),
		ip:  inventorydata.NewProcessor(l, ctx),
		chp: chalkboarddata.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) emit(f func(mb *message.Buffer) error) error {
	return message.Emit(kproducer.ProviderImpl(p.l)(p.ctx))(f)
}

// gameTypeOf maps a room type discriminator to its persisted record game type.
func gameTypeOf(roomType byte) record.GameType {
	if roomType == RoomTypeMatchCards {
		return record.GameTypeMatchCards
	}
	return record.GameTypeOmok
}

// clampPieceType bounds the piece/spec selector to its per-game valid range.
func clampPieceType(roomType byte, pieceType byte) byte {
	if roomType == RoomTypeMatchCards {
		if pieceType > matchCardsSpecMax {
			return matchCardsSpecMax
		}
		return pieceType
	}
	if pieceType > omokPieceMax {
		return omokPieceMax
	}
	return pieceType
}

// creationItemId is the item the character must possess to open the room. The
// item is never consumed.
func creationItemId(roomType byte, pieceType byte) uint32 {
	if roomType == RoomTypeMatchCards {
		return matchCardsItemId
	}
	return omokItemBase + uint32(clampPieceType(RoomTypeOmok, pieceType))
}

func (p *ProcessorImpl) Create(txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.create(mb, txId, f, characterId, roomType, title, private, password, pieceType)
	})
}

// create validates the CREATE ladder in order (dead → fieldLimit → chalkboard →
// item → already-in-room), then registers the room and emits CREATED +
// BALLOON_UPDATED. A validation failure buffers the CREATE_ERROR event and
// returns nil so the buffer still flushes.
func (p *ProcessorImpl) create(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error {
	hp, err := p.cp.Hp(characterId)
	if err != nil {
		return err
	}
	if hp == 0 {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errNotWhenDead))
	}

	fieldLimit, err := p.mp.FieldLimit(f.MapId())
	if err != nil {
		return err
	}
	if fieldLimit&0x80 != 0 {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errCannotStartGameHere))
	}

	open, err := p.chp.HasOpen(characterId)
	if err != nil {
		return err
	}
	if open {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errCannotOpenMiniRoomHere))
	}

	hasItem, err := p.ip.HasItem(characterId, creationItemId(roomType, pieceType))
	if err != nil {
		return err
	}
	if !hasItem {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errUnable))
	}

	if _, ok := p.reg.GetByMember(p.t, characterId); ok {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errUnable))
	}

	gameType := gameTypeOf(roomType)
	room := NewBuilder(roomType, characterId, f).
		SetTitle(title).
		SetPrivate(private).
		SetPassword(password).
		SetPieceType(clampPieceType(roomType, pieceType)).
		SetGameType(gameType).
		Build()

	if err := p.reg.Create(p.t, room); err != nil {
		return mb.Put(minigame.EnvEventTopicStatus, createErrorProvider(txId, f, characterId, errUnable))
	}

	ownerRecord, err := record.GetOrZero(p.db, p.t.Id(), characterId, gameType)
	if err != nil {
		return err
	}

	if err := mb.Put(minigame.EnvEventTopicStatus, createdProvider(txId, room, ownerRecord)); err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, room, 1, false))
}

func (p *ProcessorImpl) Visit(txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.visit(mb, txId, f, characterId, roomId, password)
	})
}

// visit validates the VISIT ladder in order (already-in-room → absent → full →
// password → dead → chalkboard), then seats the visitor and emits ENTERED +
// BALLOON_UPDATED. Scores reset to 0 when a different visitor than last time
// joins. The already-in-room gate (design §3.4 UNABLE convention, same as
// Create's) must run first: seating a character still indexed to another room
// would have Registry.Update re-point members[t][characterId] at the new room
// while the old room still lists them — leaking the old room if they owned it,
// or leaving a phantom visitor in it.
func (p *ProcessorImpl) visit(mb *message.Buffer, txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error {
	if _, ok := p.reg.GetByMember(p.t, characterId); ok {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errUnable))
	}

	room, ok := p.reg.Get(p.t, roomId)
	if !ok {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errRoomClosed))
	}
	if room.VisitorId() != 0 {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errFull))
	}
	if room.Private() && room.Password() != "" && !strings.EqualFold(password, room.Password()) {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errIncorrectPassword))
	}

	hp, err := p.cp.Hp(characterId)
	if err != nil {
		return err
	}
	if hp == 0 {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errNotWhenDead))
	}

	open, err := p.chp.HasOpen(characterId)
	if err != nil {
		return err
	}
	if open {
		return mb.Put(minigame.EnvEventTopicStatus, enterErrorProvider(txId, f, roomId, characterId, errCannotOpenMiniRoomHere))
	}

	updated, err := p.reg.Update(p.t, roomId, func(cur Room) (Room, error) {
		b := Clone(cur).
			SetVisitorId(characterId).
			SetLastVisitorId(characterId).
			SetVisitorReady(false)
		if characterId != cur.LastVisitorId() {
			b.SetOwnerScore(0).SetVisitorScore(0)
		}
		return b.Build(), nil
	})
	if err != nil {
		return err
	}

	ownerRecord, err := record.GetOrZero(p.db, p.t.Id(), updated.OwnerId(), updated.GameType())
	if err != nil {
		return err
	}
	visitorRecord, err := record.GetOrZero(p.db, p.t.Id(), characterId, updated.GameType())
	if err != nil {
		return err
	}

	if err := mb.Put(minigame.EnvEventTopicStatus, enteredProvider(txId, updated, ownerRecord, visitorRecord)); err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, updated, 2, false))
}

func (p *ProcessorImpl) Leave(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.leave(mb, txId, characterId)
	})
}

// leave handles a member leaving. A non-member is a no-op. The owner leaving
// closes the room (ROOM_CLOSED + balloon remove + registry Remove); the visitor
// leaving frees the slot (LEFT status 4 + balloon occupancy 1). Mid-game forfeit
// resolution arrives in Task 15.
func (p *ProcessorImpl) leave(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	slot, _ := room.SlotOf(characterId)
	if slot == 0 {
		if err := mb.Put(minigame.EnvEventTopicStatus, roomClosedProvider(txId, room, leaveStatusClosed)); err != nil {
			return err
		}
		p.reg.Remove(p.t, room.Id())
		return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, room, 1, true))
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetVisitorId(0).SetVisitorReady(false).Build(), nil
	})
	if err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, leftProvider(txId, updated, 1, leaveStatusLeft, characterId)); err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, updated, 1, false))
}

func (p *ProcessorImpl) Expel(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.expel(mb, txId, characterId)
	})
}

// expel handles the owner ejecting the visitor. Only the owner may expel, and
// only when a visitor is present; anything else is a no-op. Emits LEFT status 5
// + balloon occupancy 1. Mid-game forfeit resolution arrives in Task 15.
func (p *ProcessorImpl) expel(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if slot, _ := room.SlotOf(characterId); slot != 0 {
		return nil
	}
	visitorId := room.VisitorId()
	if visitorId == 0 {
		return nil
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetVisitorId(0).SetVisitorReady(false).Build(), nil
	})
	if err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, leftProvider(txId, updated, 1, leaveStatusExpelled, visitorId)); err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, updated, 1, false))
}

func (p *ProcessorImpl) Chat(txId uuid.UUID, f field.Model, characterId uint32, msg string) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.chat(mb, txId, characterId, msg)
	})
}

// chat rebroadcasts a member's message to the room. A non-member's chat is
// silently dropped with no event.
func (p *ProcessorImpl) chat(mb *message.Buffer, txId uuid.UUID, characterId uint32, msg string) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	slot, _ := room.SlotOf(characterId)
	return mb.Put(minigame.EnvEventTopicStatus, chatProvider(txId, room, slot, characterId, msg))
}

// TeardownCharacter releases whatever room the character occupies (map-leave /
// logout / session destroy), on the same path as an explicit LEAVE. Wired by
// Task 16.
func (p *ProcessorImpl) TeardownCharacter(characterId uint32) error {
	if _, ok := p.reg.GetByMember(p.t, characterId); !ok {
		return nil
	}
	return p.emit(func(mb *message.Buffer) error {
		return p.leave(mb, uuid.New(), characterId)
	})
}
