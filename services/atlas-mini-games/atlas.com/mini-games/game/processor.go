package game

import (
	chalkboarddata "atlas-mini-games/data/chalkboard"
	characterdata "atlas-mini-games/data/character"
	inventorydata "atlas-mini-games/data/inventory"
	mapdata "atlas-mini-games/data/map"
	"atlas-mini-games/game/matchcards"
	"atlas-mini-games/game/omok"
	"atlas-mini-games/kafka/message"
	"atlas-mini-games/kafka/message/minigame"
	kproducer "atlas-mini-games/kafka/producer"
	"atlas-mini-games/record"
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// tieCooldown is the window during which a tie result awards no session score
// (design §3.3 / Cosmic MiniGame.java:240-298, 5-minute tie-score cooldown).
const tieCooldown = 5 * time.Minute

// forfeitFarmThreshold suppresses the winner's +50 once the forfeiting loser
// already has this many forfeits this session (anti-farm, design §3.3).
const forfeitFarmThreshold byte = 4

// Session score deltas (per-room, never persisted — design §3.3 / FR-9.2).
const (
	scoreWin         int32 = 50
	scoreLoss        int32 = 15
	scoreLossForfeit int32 = -15
	scoreTie         int32 = 10
)

// Game-end result types (GameEndedEventBody.ResultType, §G5 mode-62 RESULT).
const (
	resultWin     byte = 0
	resultTie     byte = 1
	resultForfeit byte = 2
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

// Processor handles mini-game lifecycle and gameplay commands.
type Processor interface {
	Create(txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error
	Visit(txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error
	Leave(txId uuid.UUID, f field.Model, characterId uint32) error
	Chat(txId uuid.UUID, f field.Model, characterId uint32, message string) error
	Expel(txId uuid.UUID, f field.Model, characterId uint32) error
	TeardownCharacter(characterId uint32) error
	Ready(txId uuid.UUID, f field.Model, characterId uint32) error
	Unready(txId uuid.UUID, f field.Model, characterId uint32) error
	Start(txId uuid.UUID, f field.Model, characterId uint32) error
	MoveStone(txId uuid.UUID, f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) error
	FlipCard(txId uuid.UUID, f field.Model, characterId uint32, first bool, cardIndex byte) error
	RequestTie(txId uuid.UUID, f field.Model, characterId uint32) error
	AnswerTie(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error
	GiveUp(txId uuid.UUID, f field.Model, characterId uint32) error
	RequestRetreat(txId uuid.UUID, f field.Model, characterId uint32) error
	AnswerRetreat(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error
	Skip(txId uuid.UUID, f field.Model, characterId uint32) error
	ExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error
	CancelExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error
	// RoomsInField returns every mini-game room registered in field f for the
	// processor's tenant (the rooms-in-field REST read).
	RoomsInField(f field.Model) []Room
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
	// now and rng are injected so the tie cooldown and the MatchCards deck
	// shuffle are deterministic under test. Both are nil-safe: an unset field
	// falls back to time.Now / a time-seeded rand (see clock/shuffle).
	now func() time.Time
	rng *rand.Rand
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
		now: time.Now,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// clock returns the processor's injected clock, or time.Now when unset.
func (p *ProcessorImpl) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// shuffle randomizes deck using the injected rand source, or a time-seeded one
// when unset.
func (p *ProcessorImpl) shuffle(deck []uint32) {
	r := p.rng
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	matchcards.Shuffle(deck, r)
}

// stoneColor returns the 1-based Omok stone color for a player slot. Per
// ida-notes §G1 (COmokDlg::OnUserStart: m_nPlayerColor = 2 - (startByte !=
// mySlot), startByte == FirstMover): the player whose slot == FirstMover is the
// second mover with color 2; the other (the first mover) has color 1.
func stoneColor(slot byte, firstMover byte) byte {
	if slot == firstMover {
		return 2
	}
	return 1
}

func (p *ProcessorImpl) emit(f func(mb *message.Buffer) error) error {
	return message.Emit(kproducer.ProviderImpl(p.l)(p.ctx))(f)
}

// RoomsInField returns every mini-game room registered in field f for the
// processor's tenant. Callers (the REST handler) go through the processor
// rather than reaching into the registry directly (DOM-14).
func (p *ProcessorImpl) RoomsInField(f field.Model) []Room {
	return p.reg.GetInField(p.t, f)
}

// gameTypeOf maps a room type discriminator to its persisted record game type.
func gameTypeOf(roomType byte) record.GameType {
	if roomType == miniroom.MatchCards {
		return record.GameTypeMatchCards
	}
	return record.GameTypeOmok
}

// clampPieceType bounds the piece/spec selector to its per-game valid range.
func clampPieceType(roomType byte, pieceType byte) byte {
	if roomType == miniroom.MatchCards {
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
	if roomType == miniroom.MatchCards {
		return matchCardsItemId
	}
	return omokItemBase + uint32(clampPieceType(miniroom.Omok, pieceType))
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

	ownerRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), characterId, gameType)
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

	ownerRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), updated.OwnerId(), updated.GameType())
	if err != nil {
		return err
	}
	visitorRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), characterId, updated.GameType())
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
// leaving frees the slot (LEFT status 4 + balloon occupancy 1). Leaving mid-game
// forfeits the game to the opponent (design §3.3, MiniGame.java:137-156): the
// forfeit game-end resolves first, THEN the membership teardown runs.
func (p *ProcessorImpl) leave(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if room.InProgress() {
		slot, _ := room.SlotOf(characterId)
		// The leaver forfeits; the opponent (1-slot) takes the forfeit win.
		if err := p.endGame(mb, txId, room.Id(), resultForfeit, 1-slot); err != nil {
			return err
		}
		// endGame reset the room and may already have torn it down (exit-after);
		// re-resolve before the explicit teardown below.
		room, ok = p.reg.GetByMember(p.t, characterId)
		if !ok {
			return nil
		}
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
// + balloon occupancy 1. Expelling the visitor mid-game forfeits the game to the
// owner first (design §3.3), THEN tears the visitor down.
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

	if room.InProgress() {
		// The expelled visitor (slot 1) forfeits; the owner (slot 0) wins.
		if err := p.endGame(mb, txId, room.Id(), resultForfeit, 0); err != nil {
			return err
		}
		room, ok = p.reg.GetByMember(p.t, characterId)
		if !ok || room.VisitorId() != visitorId {
			return nil
		}
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
// logout / session destroy), on the same path as an explicit LEAVE — including
// the mid-game forfeit resolution in leave. Wired by Task 16.
func (p *ProcessorImpl) TeardownCharacter(characterId uint32) error {
	if _, ok := p.reg.GetByMember(p.t, characterId); !ok {
		return nil
	}
	return p.emit(func(mb *message.Buffer) error {
		return p.leave(mb, uuid.New(), characterId)
	})
}

func (p *ProcessorImpl) Ready(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.setReady(mb, txId, characterId, true)
	})
}

func (p *ProcessorImpl) Unready(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.setReady(mb, txId, characterId, false)
	})
}

// setReady toggles the visitor's ready flag. Only the visitor readies (the owner
// starts, §G5 READY/UNREADY are the visitor's button); a non-visitor is silently
// dropped (Cosmic parity). Readying mid-game is dropped. Emits READY/UNREADY.
func (p *ProcessorImpl) setReady(mb *message.Buffer, txId uuid.UUID, characterId uint32, ready bool) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if slot, _ := room.SlotOf(characterId); slot != 1 {
		return nil
	}
	if room.InProgress() {
		return nil
	}
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetVisitorReady(ready).Build(), nil
	})
	if err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, readyProvider(txId, updated, characterId, ready))
}

func (p *ProcessorImpl) Start(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.start(mb, txId, characterId)
	})
}

// start begins the game. Only the owner may start (slot 0), only when a visitor
// is present and ready, and only when no game is already running. START
// initialises the board/deck, clears the exit-after and deny-tie bits (design
// §3.3), and sets CurrentTurn to the first mover. Per §G1 the START byte is the
// FirstMover slot (the second mover) and the first move goes to the OTHER slot,
// so CurrentTurn = 1 - FirstMover. MatchCards builds and shuffles a deck of
// BuildDeck(MatchesToWin(pieceType)); Omok starts with an empty board.
func (p *ProcessorImpl) start(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if slot, _ := room.SlotOf(characterId); slot != 0 {
		return nil
	}
	if room.InProgress() || room.VisitorId() == 0 || !room.VisitorReady() {
		return nil
	}

	var deck []uint32
	if room.RoomType() == miniroom.MatchCards {
		pairs, valid := matchcards.MatchesToWin(room.PieceType())
		if !valid {
			return nil
		}
		deck = matchcards.BuildDeck(pairs)
		p.shuffle(deck)
	}

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		var empty [omok.Cells]byte
		b := Clone(cur).
			SetInProgress(true).
			SetCurrentTurn(byte(1)-cur.FirstMover()). // §G1: first mover = slot != START byte
			SetExitAfter(0, false).SetExitAfter(1, false).
			SetDeniedTie(0, false).SetDeniedTie(1, false).
			SetOwnerPairs(0).SetVisitorPairs(0).
			SetFirstSlot(-1).
			SetBoard(empty).
			SetMoves(nil).
			SetDeck(deck)
		return b.Build(), nil
	})
	if err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, startedProvider(txId, updated, deck)); err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, updated, 2, false))
}

func (p *ProcessorImpl) MoveStone(txId uuid.UUID, f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.moveStone(mb, txId, characterId, x, y)
	})
}

// moveStone places an Omok stone. The move is dropped unless it is a running
// Omok game, the sender is the current-turn player (FR-5.1 server-side turn
// validation), and the target cell is empty & in-bounds (omok.Place). The stone
// color is derived server-side from the slot (§G1), not trusted from the client.
// A valid non-winning move flips the turn; a winning move (omok.Wins) broadcasts
// STONE_PLACED first, then resolves the win via endGame (which wipes the board).
func (p *ProcessorImpl) moveStone(mb *message.Buffer, txId uuid.UUID, characterId uint32, x uint32, y uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if !room.InProgress() || room.RoomType() != miniroom.Omok {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok || slot != room.CurrentTurn() {
		return nil
	}
	color := stoneColor(slot, room.FirstMover())
	board, placed := omok.Place(room.Board(), x, y, color)
	if !placed {
		return nil
	}
	win := omok.Wins(board, x, y)

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		b := Clone(cur).SetBoard(board).SetMoves(append(cur.Moves(), Move{X: x, Y: y, Stone: color}))
		if !win {
			b.SetCurrentTurn(byte(1) - slot)
		}
		return b.Build(), nil
	})
	if err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, stonePlacedProvider(txId, updated, x, y, color, characterId)); err != nil {
		return err
	}
	if win {
		return p.endGame(mb, txId, updated.Id(), resultWin, slot)
	}
	return nil
}

func (p *ProcessorImpl) FlipCard(txId uuid.UUID, f field.Model, characterId uint32, first bool, cardIndex byte) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.flipCard(mb, txId, characterId, first, cardIndex)
	})
}

// flipCard resolves a MatchCards flip. Dropped unless it is a running MatchCards
// game, the sender is the current-turn player, and the card index is in range
// (§G5 mode-68). A first flip records the pending card index (Room.FirstSlot,
// used to compare the pair on the second flip) and broadcasts CARD_FLIPPED with
// SecondFlip=false (the channel forwards it to the opponent only, design §3.2).
// A second flip compares the two cards: a match increments that player's pair
// count and retains the turn; a mismatch passes the turn. ResultType comes from
// matchcards.FlipResultType. When every pair is matched the game ends (more
// pairs wins; equal pairs tie, design §3.2).
func (p *ProcessorImpl) flipCard(mb *message.Buffer, txId uuid.UUID, characterId uint32, first bool, cardIndex byte) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	if !room.InProgress() || room.RoomType() != miniroom.MatchCards {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok || slot != room.CurrentTurn() {
		return nil
	}
	deck := room.Deck()
	if int(cardIndex) >= len(deck) {
		return nil
	}

	if first {
		updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
			return Clone(cur).SetFirstSlot(int16(cardIndex)).Build(), nil
		})
		if err != nil {
			return err
		}
		return mb.Put(minigame.EnvEventTopicStatus, cardFlippedProvider(txId, updated, false, cardIndex, 0, 0, characterId))
	}

	firstIndex := room.FirstSlot()
	if firstIndex < 0 || int(firstIndex) >= len(deck) || byte(firstIndex) == cardIndex {
		return nil
	}
	match := deck[firstIndex] == deck[cardIndex]
	resultType := matchcards.FlipResultType(slot == 0, match)

	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		b := Clone(cur).SetFirstSlot(-1)
		if match {
			if slot == 0 {
				b.SetOwnerPairs(cur.OwnerPairs() + 1)
			} else {
				b.SetVisitorPairs(cur.VisitorPairs() + 1)
			}
			// match retains the turn (CurrentTurn unchanged)
		} else {
			b.SetCurrentTurn(byte(1) - slot)
		}
		return b.Build(), nil
	})
	if err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, cardFlippedProvider(txId, updated, true, cardIndex, byte(firstIndex), resultType, characterId)); err != nil {
		return err
	}

	matchesToWin, _ := matchcards.MatchesToWin(updated.PieceType())
	if updated.OwnerPairs()+updated.VisitorPairs() != matchesToWin {
		return nil
	}
	if updated.OwnerPairs() > updated.VisitorPairs() {
		return p.endGame(mb, txId, updated.Id(), resultWin, 0)
	}
	if updated.VisitorPairs() > updated.OwnerPairs() {
		return p.endGame(mb, txId, updated.Id(), resultWin, 1)
	}
	return p.endGame(mb, txId, updated.Id(), resultTie, 0)
}

func (p *ProcessorImpl) RequestTie(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.requestTie(mb, txId, characterId)
	})
}

// requestTie forwards a tie proposal to the opponent, but only for a running
// game and only if the requester has not already been denied this game (deny
// bits, design §3.3 / MiniGame.java:220-238). The channel targets the event at
// the opponent (CharacterId == requester).
func (p *ProcessorImpl) requestTie(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok || room.DeniedTie(slot) {
		return nil
	}
	return mb.Put(minigame.EnvEventTopicStatus, tieRequestedProvider(txId, room, characterId))
}

func (p *ProcessorImpl) AnswerTie(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.answerTie(mb, txId, characterId, accept)
	})
}

// answerTie resolves a tie proposal. Accept ends the game as a tie. Decline sets
// the requester's deny bit — the requester (opponent of the answerer) can no
// longer propose a tie this game — and forwards TIE_ANSWERED{Accept:false} to
// the requester (design §3.3).
func (p *ProcessorImpl) answerTie(mb *message.Buffer, txId uuid.UUID, characterId uint32, accept bool) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	if accept {
		return p.endGame(mb, txId, room.Id(), resultTie, 0)
	}
	requesterSlot := byte(1) - slot
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetDeniedTie(requesterSlot, true).Build(), nil
	})
	if err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, tieAnsweredProvider(txId, updated, characterId, false))
}

func (p *ProcessorImpl) GiveUp(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.giveUp(mb, txId, characterId)
	})
}

// giveUp forfeits the running game: the sender loses, the opponent takes the
// forfeit win (ResultType 2, design §3.3 / PlayerInteractionHandler.java:411-425).
func (p *ProcessorImpl) giveUp(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	return p.endGame(mb, txId, room.Id(), resultForfeit, byte(1)-slot)
}

func (p *ProcessorImpl) RequestRetreat(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.requestRetreat(mb, txId, characterId)
	})
}

// requestRetreat forwards an undo request to the opponent (Omok only, §G2). Per
// the client send gate (COmokDlg::SendRetreatRequest) it is only sent when the
// requester has just placed a stone; we mirror that by requiring the tail move
// to be the requester's own stone. The channel targets the event at the opponent
// (CharacterId == requester).
func (p *ProcessorImpl) requestRetreat(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() || room.RoomType() != miniroom.Omok {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	moves := room.Moves()
	if len(moves) == 0 || moves[len(moves)-1].Stone != stoneColor(slot, room.FirstMover()) {
		return nil
	}
	return mb.Put(minigame.EnvEventTopicStatus, retreatRequestedProvider(txId, room, characterId))
}

func (p *ProcessorImpl) AnswerRetreat(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.answerRetreat(mb, txId, characterId, accept)
	})
}

// answerRetreat resolves an undo request (Omok only). The answerer is the
// opponent of the requester, so the requester = 1 - answererSlot. Accept pops the
// single most-recent stone from the board and move history and hands the turn
// back to the requester so they replay (§G2: the server chooses N stones to pop
// and the follow-up turn slot; we pop N=1 and set turn = requester, a layout the
// client honours verbatim). Decline forwards RETREAT_ANSWERED{Accept:false}.
func (p *ProcessorImpl) answerRetreat(mb *message.Buffer, txId uuid.UUID, characterId uint32, accept bool) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() || room.RoomType() != miniroom.Omok {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	if !accept {
		return mb.Put(minigame.EnvEventTopicStatus, retreatAnsweredProvider(txId, room, characterId, false))
	}
	requesterSlot := byte(1) - slot
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		b := Clone(cur)
		moves := cur.Moves()
		board := cur.Board()
		if len(moves) > 0 {
			m := moves[len(moves)-1]
			board[int(m.Y)*omok.BoardSize+int(m.X)] = 0
			moves = moves[:len(moves)-1]
		}
		return b.SetBoard(board).SetMoves(moves).SetCurrentTurn(requesterSlot).Build(), nil
	})
	if err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, retreatAnsweredProvider(txId, updated, characterId, true))
}

func (p *ProcessorImpl) Skip(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.skip(mb, txId, characterId)
	})
}

// skip yields the current player's turn to the opponent (Atlas server-side turn
// tracking, design §3.1). SKIPPED.Who carries the NEXT-mover slot (== 1 - skipper
// == new CurrentTurn): owner-skip emits 1, visitor-skip emits 0, matching Cosmic
// getMiniGameSkipOwner(0x01)/getMiniGameSkipVisitor(0x00) read as "next mover"
// per ida-notes §G5.
func (p *ProcessorImpl) skip(mb *message.Buffer, txId uuid.UUID, characterId uint32) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok || !room.InProgress() {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	next := byte(1) - slot
	updated, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetCurrentTurn(next).Build(), nil
	})
	if err != nil {
		return err
	}
	return mb.Put(minigame.EnvEventTopicStatus, skippedProvider(txId, updated, next, characterId))
}

func (p *ProcessorImpl) ExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.setExitAfter(mb, txId, characterId, true)
	})
}

func (p *ProcessorImpl) CancelExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error {
	return p.emit(func(mb *message.Buffer) error {
		return p.setExitAfter(mb, txId, characterId, false)
	})
}

// setExitAfter sets or clears the member's exit-after-game flag (design §3.3:
// settable any time, cleared on START, honored at game end). It emits no event —
// the flag is a local UI toggle whose only observable effect is the deferred
// leave at game end.
func (p *ProcessorImpl) setExitAfter(mb *message.Buffer, txId uuid.UUID, characterId uint32, exit bool) error {
	room, ok := p.reg.GetByMember(p.t, characterId)
	if !ok {
		return nil
	}
	slot, ok := room.SlotOf(characterId)
	if !ok {
		return nil
	}
	_, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetExitAfter(slot, exit).Build(), nil
	})
	return err
}

// endGame is the single game-end transition shared by every resolution path
// (win via move/flip, tie via ANSWER_TIE, forfeit via GIVE_UP or mid-game
// leave/expel/teardown). It is idempotent: a room that is not InProgress is a
// no-op, mirroring Cosmic minigameMatchFinish's double-resolution guard
// (design §3.3). resultType is 0 win / 1 tie / 2 forfeit; winnerSlot is the
// winning slot (ignored for a tie).
//
// Order: record.ApplyResult — the durable commit — runs FIRST, before the
// registry swap and before any event is emitted (design §2). If it fails the
// room is left untouched (still InProgress, so a retry/re-trigger can still
// resolve the game), nothing is emitted, and the error is returned; swapping
// first would strand a permanently-!InProgress room whose record was never
// persisted, with the error swallowed by the fire-and-forget handler. After
// the commit succeeds: compute session scores + forfeit counters and reset the
// room for a rematch (board/deck/pairs/firstSlot/deny bits/inProgress/
// currentTurn cleared; session scores + FirstMover kept, FirstMover advanced
// to the winner on a non-tie per Cosmic setPiece), swap it under the registry
// lock, emit GAME_ENDED (refreshed records + scores) and BALLOON_UPDATED{
// InProgress:false}, and finally honor the exit-after flags by processing that
// side's leave after the result.
func (p *ProcessorImpl) endGame(mb *message.Buffer, txId uuid.UUID, roomId uint32, resultType byte, winnerSlot byte) error {
	room, ok := p.reg.Get(p.t, roomId)
	if !ok || !room.InProgress() {
		return nil
	}
	ownerId := room.OwnerId()
	visitorId := room.VisitorId()
	gameType := room.GameType()
	ownerExit := room.ExitAfter(0)
	visitorExit := room.ExitAfter(1)
	tie := resultType == resultTie
	now := p.clock()

	// ApplyResult commits before the registry swap and before any event is
	// emitted (design §2). On failure the room stays InProgress, untouched.
	if err := record.ApplyResult(p.db.WithContext(p.ctx), gameType, ownerId, visitorId, winnerSlot, tie); err != nil {
		return err
	}

	updated, err := p.reg.Update(p.t, roomId, func(cur Room) (Room, error) {
		return resolvedRoom(cur, resultType, winnerSlot, now), nil
	})
	if err != nil {
		return err
	}
	ownerRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), ownerId, gameType)
	if err != nil {
		return err
	}
	visitorRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), visitorId, gameType)
	if err != nil {
		return err
	}

	if err := mb.Put(minigame.EnvEventTopicStatus, gameEndedProvider(txId, updated, resultType, winnerSlot, ownerRecord, visitorRecord)); err != nil {
		return err
	}
	if err := mb.Put(minigame.EnvEventTopicStatus, balloonProvider(txId, updated, 2, false)); err != nil {
		return err
	}

	// Honor exit-after: process that side's leave after the result. The owner's
	// leave closes the room (visitor's exit is then moot); otherwise the visitor
	// leaves. leave sees InProgress=false now, so it will not re-forfeit.
	if ownerExit {
		return p.leave(mb, txId, ownerId)
	}
	if visitorExit {
		return p.leave(mb, txId, visitorId)
	}
	return nil
}

// resolvedRoom computes the post-game Room: session scores + forfeit counters
// per design §3.3, FirstMover advanced to the winner on a non-tie (Cosmic
// setPiece: 0 after an owner win, 1 after a visitor win), and every per-game
// field reset for the rematch. Session scores, FirstMover, VisitorId, and the
// forfeit counters persist; VisitorReady resets so both sides must ready up
// again before the next START.
func resolvedRoom(cur Room, resultType byte, winnerSlot byte, now time.Time) Room {
	b := Clone(cur)

	ownerScore := cur.OwnerScore()
	visitorScore := cur.VisitorScore()
	ownerForfeits := cur.OwnerForfeits()
	visitorForfeits := cur.VisitorForfeits()

	switch resultType {
	case resultTie:
		// Tie awards +10 each only outside the 5-minute cooldown (design §3.3).
		if !now.Before(cur.TieCooldownUntil()) {
			ownerScore += scoreTie
			visitorScore += scoreTie
			b.SetTieCooldownUntil(now.Add(tieCooldown))
		}
		// FirstMover unchanged on a tie (no winner).
	case resultForfeit:
		// The loser (1 - winnerSlot) forfeited: -15 and forfeit counter++; the
		// winner's +50 is suppressed once the loser already had >= 4 forfeits.
		if winnerSlot == 0 {
			if visitorForfeits < forfeitFarmThreshold {
				ownerScore += scoreWin
			}
			visitorScore += scoreLossForfeit
			visitorForfeits++
		} else {
			if ownerForfeits < forfeitFarmThreshold {
				visitorScore += scoreWin
			}
			ownerScore += scoreLossForfeit
			ownerForfeits++
		}
		b.SetFirstMover(winnerSlot)
	default: // resultWin
		if winnerSlot == 0 {
			ownerScore += scoreWin
			visitorScore += scoreLoss
		} else {
			visitorScore += scoreWin
			ownerScore += scoreLoss
		}
		b.SetFirstMover(winnerSlot)
	}

	var empty [omok.Cells]byte
	return b.
		SetInProgress(false).
		SetVisitorReady(false).
		SetCurrentTurn(0).
		SetBoard(empty).
		SetMoves(nil).
		SetDeck(nil).
		SetOwnerPairs(0).
		SetVisitorPairs(0).
		SetFirstSlot(-1).
		SetDeniedTie(0, false).
		SetDeniedTie(1, false).
		SetExitAfter(0, false).
		SetExitAfter(1, false).
		SetOwnerScore(ownerScore).
		SetVisitorScore(visitorScore).
		SetOwnerForfeits(ownerForfeits).
		SetVisitorForfeits(visitorForfeits).
		Build()
}
