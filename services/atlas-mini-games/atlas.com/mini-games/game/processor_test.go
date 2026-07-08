package game

import (
	"atlas-mini-games/game/omok"
	"atlas-mini-games/kafka/message"
	"atlas-mini-games/kafka/message/minigame"
	"atlas-mini-games/record"
	"context"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- fakes for the four injected REST-client seams ---------------------------

type fakeCharacter struct {
	hp  uint16
	err error
}

func (f fakeCharacter) Hp(uint32) (uint16, error) { return f.hp, f.err }

type fakeMap struct {
	fieldLimit uint32
	err        error
}

func (f fakeMap) FieldLimit(_map.Id) (uint32, error) { return f.fieldLimit, f.err }

type fakeInventory struct {
	has       bool
	err       error
	lastItem  uint32
	lastOwner uint32
}

func (f *fakeInventory) HasItem(characterId uint32, itemId uint32) (bool, error) {
	f.lastItem = itemId
	f.lastOwner = characterId
	return f.has, f.err
}

type fakeChalkboard struct {
	open bool
	err  error
}

func (f fakeChalkboard) HasOpen(uint32) (bool, error) { return f.open, f.err }

// --- test scaffolding --------------------------------------------------------

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	database.RegisterTenantCallbacks(l, db)

	err = db.Exec(`
		CREATE TABLE game_records (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			game_type TEXT NOT NULL,
			wins INTEGER NOT NULL DEFAULT 0,
			ties INTEGER NOT NULL DEFAULT 0,
			losses INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(tenant_id, character_id, game_type)
		)
	`).Error
	require.NoError(t, err)
	return db
}

type harness struct {
	p     *ProcessorImpl
	t     tenant.Model
	f     field.Model
	cp    fakeCharacter
	mp    fakeMap
	ip    *fakeInventory
	chp   fakeChalkboard
	clock time.Time
}

// newHarness builds a processor with all validation gates open (alive, field ok,
// no chalkboard, has item) and a fresh tenant so registry state is isolated.
func newHarness(t *testing.T) *harness {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	h := &harness{
		t:   ten,
		f:   field.NewBuilder(1, 1, 100000).Build(),
		cp:  fakeCharacter{hp: 100},
		mp:  fakeMap{fieldLimit: 0},
		ip:  &fakeInventory{has: true},
		chp: fakeChalkboard{open: false},
	}
	h.p = &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  setupTestDB(t),
		t:   ten,
		reg: GetRegistry(),
		cp:  h.cp,
		mp:  h.mp,
		ip:  h.ip,
		chp: h.chp,
		// Deterministic clock (advanceable via h.clock) and rand for the tie
		// cooldown and MatchCards deck shuffle.
		now: func() time.Time { return h.clock },
		rng: rand.New(rand.NewSource(1)),
	}
	h.clock = time.Unix(1_600_000_000, 0).UTC()
	return h
}

// rebuild re-wires the processor's client seams after a test mutates the fakes.
func (h *harness) rebuild() {
	h.p.cp = h.cp
	h.p.mp = h.mp
	h.p.ip = h.ip
	h.p.chp = h.chp
}

func decodeEvents[E any](t *testing.T, buf *message.Buffer, eventType string) []minigame.StatusEvent[E] {
	t.Helper()
	var out []minigame.StatusEvent[E]
	for _, m := range buf.GetAll()[minigame.EnvEventTopicStatus] {
		var probe struct {
			Type string `json:"type"`
		}
		require.NoError(t, json.Unmarshal(m.Value, &probe))
		if probe.Type != eventType {
			continue
		}
		var ev minigame.StatusEvent[E]
		require.NoError(t, json.Unmarshal(m.Value, &ev))
		out = append(out, ev)
	}
	return out
}

func requireOneError(t *testing.T, buf *message.Buffer, eventType string, code string) {
	t.Helper()
	evs := decodeEvents[minigame.ErrorEventBody](t, buf, eventType)
	require.Len(t, evs, 1, "expected exactly one %s event", eventType)
	assert.Equal(t, code, evs[0].Body.Code)
}

// --- helpers to derive the pure item-id logic --------------------------------

func TestCreationItemId(t *testing.T) {
	assert.Equal(t, uint32(4080000), creationItemId(RoomTypeOmok, 0))
	assert.Equal(t, uint32(4080005), creationItemId(RoomTypeOmok, 5))
	assert.Equal(t, uint32(4080011), creationItemId(RoomTypeOmok, 11))
	// pieceType clamps to [0,11] for omok.
	assert.Equal(t, uint32(4080011), creationItemId(RoomTypeOmok, 99))
	// match cards always uses the single set id regardless of spec.
	assert.Equal(t, uint32(4080100), creationItemId(RoomTypeMatchCards, 0))
	assert.Equal(t, uint32(4080100), creationItemId(RoomTypeMatchCards, 2))
	assert.Equal(t, uint32(4080100), creationItemId(RoomTypeMatchCards, 99))
}

func TestClampPieceType(t *testing.T) {
	assert.Equal(t, byte(11), clampPieceType(RoomTypeOmok, 50))
	assert.Equal(t, byte(7), clampPieceType(RoomTypeOmok, 7))
	assert.Equal(t, byte(2), clampPieceType(RoomTypeMatchCards, 9))
	assert.Equal(t, byte(1), clampPieceType(RoomTypeMatchCards, 1))
}

// --- CREATE ------------------------------------------------------------------

func TestCreate_HappyPath(t *testing.T) {
	h := newHarness(t)
	owner := uint32(1001)

	buf := message.NewBuffer()
	require.NoError(t, h.p.create(buf, uuid.New(), h.f, owner, RoomTypeOmok, "hi", false, "", 3))

	// Room registered, keyed by owner id (D2).
	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok)
	assert.Equal(t, owner, r.OwnerId())
	assert.Equal(t, byte(3), r.PieceType())

	// Item was checked but not consumed (fake never decrements); omok id 4080003.
	assert.Equal(t, uint32(4080003), h.ip.lastItem)

	created := decodeEvents[minigame.CreatedEventBody](t, buf, minigame.EventTypeCreated)
	require.Len(t, created, 1)
	assert.Equal(t, owner, created[0].RoomId)
	assert.Equal(t, owner, created[0].OwnerId)
	assert.Equal(t, "hi", created[0].Body.Title)
	assert.Equal(t, "OMOK", created[0].Body.OwnerRecord.GameType)
	assert.Equal(t, uint32(0), created[0].Body.OwnerRecord.Wins)

	balloon := decodeEvents[minigame.BalloonEventBody](t, buf, minigame.EventTypeBalloonUpdated)
	require.Len(t, balloon, 1)
	assert.Equal(t, byte(1), balloon[0].Body.Occupancy)
	assert.False(t, balloon[0].Body.Remove)
}

func TestCreate_ValidationLadder(t *testing.T) {
	owner := uint32(2002)

	tests := []struct {
		name    string
		mutate  func(h *harness)
		preSeed bool
		code    string
	}{
		{
			name:   "dead",
			mutate: func(h *harness) { h.cp = fakeCharacter{hp: 0} },
			code:   "NOT_WHEN_DEAD",
		},
		{
			name:   "field limit forbids",
			mutate: func(h *harness) { h.mp = fakeMap{fieldLimit: 0x80} },
			code:   "CANNOT_START_GAME_HERE",
		},
		{
			name:   "chalkboard open",
			mutate: func(h *harness) { h.chp = fakeChalkboard{open: true} },
			code:   "CANNOT_OPEN_MINI_ROOM_HERE",
		},
		{
			name:   "missing item",
			mutate: func(h *harness) { h.ip = &fakeInventory{has: false} },
			code:   "UNABLE",
		},
		{
			name:    "already in a room",
			mutate:  func(h *harness) {},
			preSeed: true,
			code:    "UNABLE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.mutate(h)
			h.rebuild()

			if tc.preSeed {
				require.NoError(t, h.p.reg.Create(h.t, NewBuilder(RoomTypeOmok, owner, h.f).Build()))
			}

			buf := message.NewBuffer()
			require.NoError(t, h.p.create(buf, uuid.New(), h.f, owner, RoomTypeOmok, "t", false, "", 0))

			requireOneError(t, buf, minigame.EventTypeCreateError, tc.code)
			assert.Empty(t, decodeEvents[minigame.CreatedEventBody](t, buf, minigame.EventTypeCreated))
		})
	}
}

// --- VISIT -------------------------------------------------------------------

func seedRoom(t *testing.T, h *harness, b *Builder) Room {
	t.Helper()
	r := b.Build()
	require.NoError(t, h.p.reg.Create(h.t, r))
	return r
}

func TestVisit_HappyPath(t *testing.T) {
	h := newHarness(t)
	owner := uint32(3001)
	visitor := uint32(3002)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetTitle("room").SetGameType("OMOK"))

	buf := message.NewBuffer()
	require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, ""))

	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok)
	assert.Equal(t, visitor, r.VisitorId())
	assert.Equal(t, visitor, r.LastVisitorId())

	entered := decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered)
	require.Len(t, entered, 1)
	assert.Equal(t, byte(1), entered[0].Body.Slot)
	assert.Equal(t, owner, entered[0].OwnerId)
	assert.Equal(t, visitor, entered[0].VisitorId)
	assert.Equal(t, "OMOK", entered[0].Body.OwnerRecord.GameType)
	assert.Equal(t, "OMOK", entered[0].Body.VisitorRecord.GameType)

	balloon := decodeEvents[minigame.BalloonEventBody](t, buf, minigame.EventTypeBalloonUpdated)
	require.Len(t, balloon, 1)
	assert.Equal(t, byte(2), balloon[0].Body.Occupancy)
}

func TestVisit_ScoreResetOnNewVisitor(t *testing.T) {
	h := newHarness(t)
	owner := uint32(3101)
	visitor := uint32(3102)
	// Previous visitor was someone else, with lingering scores.
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).
		SetGameType("OMOK").
		SetLastVisitorId(9999).
		SetOwnerScore(50).
		SetVisitorScore(15))

	buf := message.NewBuffer()
	require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, ""))

	entered := decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered)
	require.Len(t, entered, 1)
	assert.Equal(t, int32(0), entered[0].Body.OwnerScore)
	assert.Equal(t, int32(0), entered[0].Body.VisitorScore)
}

func TestVisit_ScoreRetainedOnSameVisitor(t *testing.T) {
	h := newHarness(t)
	owner := uint32(3201)
	visitor := uint32(3202)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).
		SetGameType("OMOK").
		SetLastVisitorId(visitor).
		SetOwnerScore(50).
		SetVisitorScore(15))

	buf := message.NewBuffer()
	require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, ""))

	entered := decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered)
	require.Len(t, entered, 1)
	assert.Equal(t, int32(50), entered[0].Body.OwnerScore)
	assert.Equal(t, int32(15), entered[0].Body.VisitorScore)
}

func TestVisit_ValidationLadder(t *testing.T) {
	owner := uint32(4001)
	visitor := uint32(4002)

	tests := []struct {
		name   string
		seed   func(t *testing.T, h *harness)
		mutate func(h *harness)
		pass   string
		code   string
	}{
		{
			name: "room absent",
			seed: func(t *testing.T, h *harness) {},
			code: "ROOM_CLOSED",
		},
		{
			name: "room full",
			seed: func(t *testing.T, h *harness) {
				seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(7777))
			},
			code: "FULL",
		},
		{
			name: "wrong password",
			seed: func(t *testing.T, h *harness) {
				seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetPrivate(true).SetPassword("secret"))
			},
			pass: "wrong",
			code: "INCORRECT_PASSWORD",
		},
		{
			name: "dead",
			seed: func(t *testing.T, h *harness) {
				seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK"))
			},
			mutate: func(h *harness) { h.cp = fakeCharacter{hp: 0} },
			code:   "NOT_WHEN_DEAD",
		},
		{
			name: "chalkboard open",
			seed: func(t *testing.T, h *harness) {
				seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK"))
			},
			mutate: func(h *harness) { h.chp = fakeChalkboard{open: true} },
			code:   "CANNOT_OPEN_MINI_ROOM_HERE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.seed(t, h)
			if tc.mutate != nil {
				tc.mutate(h)
				h.rebuild()
			}

			buf := message.NewBuffer()
			require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, tc.pass))

			requireOneError(t, buf, minigame.EventTypeEnterError, tc.code)
			assert.Empty(t, decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered))
		})
	}
}

func TestVisit_PasswordCases(t *testing.T) {
	owner := uint32(4101)
	visitor := uint32(4102)

	t.Run("empty password always passes", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetPrivate(true).SetPassword(""))
		buf := message.NewBuffer()
		require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, "anything"))
		require.Len(t, decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered), 1)
	})

	t.Run("case-insensitive match passes", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetPrivate(true).SetPassword("Secret"))
		buf := message.NewBuffer()
		require.NoError(t, h.p.visit(buf, uuid.New(), h.f, visitor, owner, "sEcReT"))
		require.Len(t, decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered), 1)
	})
}

// TestVisit_AlreadySeatedRejected guards the member-index corruption defect:
// letting a character seated in room A visit room B would have Registry.Update
// re-point members[t][C] from A to B while rooms[t][A] still lists C — leaking
// A permanently if C owned it, or leaving a phantom visitor if C visited it.
func TestVisit_AlreadySeatedRejected(t *testing.T) {
	ownerA := uint32(4201)
	ownerB := uint32(4202)

	t.Run("visitor of A visiting B", func(t *testing.T) {
		h := newHarness(t)
		seated := uint32(4203)
		seedRoom(t, h, NewBuilder(RoomTypeOmok, ownerA, h.f).SetGameType("OMOK").SetVisitorId(seated).SetLastVisitorId(seated))
		seedRoom(t, h, NewBuilder(RoomTypeOmok, ownerB, h.f).SetGameType("OMOK"))

		buf := message.NewBuffer()
		require.NoError(t, h.p.visit(buf, uuid.New(), h.f, seated, ownerB, ""))

		requireOneError(t, buf, minigame.EventTypeEnterError, "UNABLE")
		assert.Empty(t, decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered))

		// A's membership index is intact: the character still resolves to A.
		got, ok := h.p.reg.GetByMember(h.t, seated)
		require.True(t, ok)
		assert.Equal(t, ownerA, got.Id())
		// B remains empty.
		b, ok := h.p.reg.Get(h.t, ownerB)
		require.True(t, ok)
		assert.Equal(t, uint32(0), b.VisitorId())
	})

	t.Run("owner of A visiting B", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, NewBuilder(RoomTypeOmok, ownerA, h.f).SetGameType("OMOK"))
		seedRoom(t, h, NewBuilder(RoomTypeOmok, ownerB, h.f).SetGameType("OMOK"))

		buf := message.NewBuffer()
		require.NoError(t, h.p.visit(buf, uuid.New(), h.f, ownerA, ownerB, ""))

		requireOneError(t, buf, minigame.EventTypeEnterError, "UNABLE")
		assert.Empty(t, decodeEvents[minigame.EnteredEventBody](t, buf, minigame.EventTypeEntered))

		// A is still reachable via its owner — no leak.
		got, ok := h.p.reg.GetByMember(h.t, ownerA)
		require.True(t, ok)
		assert.Equal(t, ownerA, got.Id())
		// B remains empty.
		b, ok := h.p.reg.Get(h.t, ownerB)
		require.True(t, ok)
		assert.Equal(t, uint32(0), b.VisitorId())
	})
}

// --- LEAVE / EXPEL -----------------------------------------------------------

func TestLeave_Visitor(t *testing.T) {
	h := newHarness(t)
	owner := uint32(5001)
	visitor := uint32(5002)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(visitor).SetLastVisitorId(visitor))

	buf := message.NewBuffer()
	require.NoError(t, h.p.leave(buf, uuid.New(), visitor))

	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok)
	assert.Equal(t, uint32(0), r.VisitorId())

	left := decodeEvents[minigame.LeftEventBody](t, buf, minigame.EventTypeLeft)
	require.Len(t, left, 1)
	assert.Equal(t, byte(1), left[0].Body.Slot)
	assert.Equal(t, byte(4), left[0].Body.Status)

	balloon := decodeEvents[minigame.BalloonEventBody](t, buf, minigame.EventTypeBalloonUpdated)
	require.Len(t, balloon, 1)
	assert.Equal(t, byte(1), balloon[0].Body.Occupancy)
	assert.False(t, balloon[0].Body.Remove)
}

func TestExpel_PreGame(t *testing.T) {
	h := newHarness(t)
	owner := uint32(5101)
	visitor := uint32(5102)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(visitor).SetLastVisitorId(visitor))

	buf := message.NewBuffer()
	require.NoError(t, h.p.expel(buf, uuid.New(), owner))

	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok)
	assert.Equal(t, uint32(0), r.VisitorId())

	left := decodeEvents[minigame.LeftEventBody](t, buf, minigame.EventTypeLeft)
	require.Len(t, left, 1)
	assert.Equal(t, byte(1), left[0].Body.Slot)
	assert.Equal(t, byte(5), left[0].Body.Status)
	assert.Equal(t, visitor, left[0].VisitorId)
}

func TestExpel_NoVisitorIsNoOp(t *testing.T) {
	h := newHarness(t)
	owner := uint32(5201)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK"))

	buf := message.NewBuffer()
	require.NoError(t, h.p.expel(buf, uuid.New(), owner))
	assert.Empty(t, buf.GetAll())
}

func TestLeave_OwnerClosesRoom(t *testing.T) {
	h := newHarness(t)
	owner := uint32(6001)
	visitor := uint32(6002)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(visitor).SetLastVisitorId(visitor))

	buf := message.NewBuffer()
	require.NoError(t, h.p.leave(buf, uuid.New(), owner))

	_, ok := h.p.reg.Get(h.t, owner)
	assert.False(t, ok, "owner leaving must remove the room")

	closed := decodeEvents[minigame.RoomClosedEventBody](t, buf, minigame.EventTypeRoomClosed)
	require.Len(t, closed, 1)
	assert.Equal(t, byte(3), closed[0].Body.VisitorStatus)
	assert.Equal(t, visitor, closed[0].VisitorId)

	balloon := decodeEvents[minigame.BalloonEventBody](t, buf, minigame.EventTypeBalloonUpdated)
	require.Len(t, balloon, 1)
	assert.True(t, balloon[0].Body.Remove)
}

func TestLeave_NonMemberIsNoOp(t *testing.T) {
	h := newHarness(t)
	buf := message.NewBuffer()
	require.NoError(t, h.p.leave(buf, uuid.New(), 424242))
	assert.Empty(t, buf.GetAll())
}

// --- CHAT --------------------------------------------------------------------

func TestChat_Member(t *testing.T) {
	h := newHarness(t)
	owner := uint32(7001)
	visitor := uint32(7002)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(visitor).SetLastVisitorId(visitor))

	buf := message.NewBuffer()
	require.NoError(t, h.p.chat(buf, uuid.New(), visitor, "gg"))

	chat := decodeEvents[minigame.ChatEventBody](t, buf, minigame.EventTypeChat)
	require.Len(t, chat, 1)
	assert.Equal(t, byte(1), chat[0].Body.Slot)
	assert.Equal(t, "gg", chat[0].Body.Message)
	assert.Equal(t, visitor, chat[0].CharacterId)
}

func TestChat_NonMemberDropped(t *testing.T) {
	h := newHarness(t)
	owner := uint32(7101)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK"))

	buf := message.NewBuffer()
	require.NoError(t, h.p.chat(buf, uuid.New(), 999999, "spam"))
	assert.Empty(t, buf.GetAll())
}

// --- gameplay helpers --------------------------------------------------------

// seedIdleOmok seeds an Omok room with a seated (but not-started) visitor.
func seedIdleOmok(t *testing.T, h *harness, owner, visitor uint32) Room {
	t.Helper()
	return seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).
		SetGameType("OMOK").
		SetVisitorId(visitor).SetLastVisitorId(visitor))
}

func runningOmokBuilder(h *harness, owner, visitor uint32, firstMover byte) *Builder {
	return NewBuilder(RoomTypeOmok, owner, h.f).
		SetGameType("OMOK").
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		SetVisitorReady(true).
		SetInProgress(true).
		SetFirstMover(firstMover).
		SetCurrentTurn(byte(1) - firstMover)
}

func runningMatchCardsBuilder(h *harness, owner, visitor uint32, pieceType byte, deck []uint32) *Builder {
	return NewBuilder(RoomTypeMatchCards, owner, h.f).
		SetGameType("MATCH_CARDS").
		SetPieceType(pieceType).
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		SetVisitorReady(true).
		SetInProgress(true).
		SetFirstMover(1).
		SetCurrentTurn(0).
		SetDeck(deck)
}

// --- READY / UNREADY ---------------------------------------------------------

func TestReady_VisitorOnly(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8001)
	visitor := uint32(8002)
	seedIdleOmok(t, h, owner, visitor)

	buf := message.NewBuffer()
	require.NoError(t, h.p.setReady(buf, uuid.New(), visitor, true))

	r, _ := h.p.reg.Get(h.t, owner)
	assert.True(t, r.VisitorReady())
	require.Len(t, decodeEvents[minigame.EmptyEventBody](t, buf, minigame.EventTypeReady), 1)

	// Unready toggles it back.
	buf = message.NewBuffer()
	require.NoError(t, h.p.setReady(buf, uuid.New(), visitor, false))
	r, _ = h.p.reg.Get(h.t, owner)
	assert.False(t, r.VisitorReady())
	require.Len(t, decodeEvents[minigame.EmptyEventBody](t, buf, minigame.EventTypeUnready), 1)
}

func TestReady_OwnerAndNonMemberDropped(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8101)
	visitor := uint32(8102)
	seedIdleOmok(t, h, owner, visitor)

	buf := message.NewBuffer()
	require.NoError(t, h.p.setReady(buf, uuid.New(), owner, true)) // owner cannot ready
	require.NoError(t, h.p.setReady(buf, uuid.New(), 999999, true))
	assert.Empty(t, buf.GetAll())
	r, _ := h.p.reg.Get(h.t, owner)
	assert.False(t, r.VisitorReady())
}

// --- START -------------------------------------------------------------------

func TestStart_Omok(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8201)
	visitor := uint32(8202)
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).
		SetGameType("OMOK").
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		SetVisitorReady(true).
		SetFirstMover(1).
		SetExitAfter(0, true).
		SetDeniedTie(1, true))

	buf := message.NewBuffer()
	require.NoError(t, h.p.start(buf, uuid.New(), owner))

	r, _ := h.p.reg.Get(h.t, owner)
	assert.True(t, r.InProgress())
	assert.Equal(t, byte(0), r.CurrentTurn(), "first mover = slot != FirstMover byte (§G1)")
	assert.False(t, r.ExitAfter(0), "START clears exit-after")
	assert.False(t, r.DeniedTie(1), "START clears deny-tie bits")

	started := decodeEvents[minigame.StartedEventBody](t, buf, minigame.EventTypeStarted)
	require.Len(t, started, 1)
	assert.Equal(t, byte(1), started[0].Body.FirstMover)
	assert.Empty(t, started[0].Body.Deck, "omok deck is empty")

	balloon := decodeEvents[minigame.BalloonEventBody](t, buf, minigame.EventTypeBalloonUpdated)
	require.Len(t, balloon, 1)
	assert.True(t, balloon[0].Body.InProgress)
}

func TestStart_MatchCards(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8301)
	visitor := uint32(8302)
	seedRoom(t, h, NewBuilder(RoomTypeMatchCards, owner, h.f).
		SetGameType("MATCH_CARDS").
		SetPieceType(0). // MatchesToWin(0) == 6 pairs -> 12 cards
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		SetVisitorReady(true))

	buf := message.NewBuffer()
	require.NoError(t, h.p.start(buf, uuid.New(), owner))

	started := decodeEvents[minigame.StartedEventBody](t, buf, minigame.EventTypeStarted)
	require.Len(t, started, 1)
	require.Len(t, started[0].Body.Deck, 12, "deck = BuildDeck(MatchesToWin(0)) = 12 cards")
	counts := map[uint32]int{}
	for _, id := range started[0].Body.Deck {
		counts[id]++
	}
	assert.Len(t, counts, 6, "6 distinct pair ids")
	for id, c := range counts {
		assert.Equal(t, 2, c, "id %d appears exactly twice", id)
	}

	r, _ := h.p.reg.Get(h.t, owner)
	require.Len(t, r.Deck(), 12)
}

func TestStart_Guards(t *testing.T) {
	owner := uint32(8401)
	visitor := uint32(8402)

	tests := []struct {
		name  string
		build func(h *harness) *Builder
		by    func(owner, visitor uint32) uint32
	}{
		{
			name:  "not owner",
			build: func(h *harness) *Builder { return runningOmokBuilder(h, owner, visitor, 1).SetInProgress(false) },
			by:    func(o, v uint32) uint32 { return v },
		},
		{
			name: "visitor not ready",
			build: func(h *harness) *Builder {
				return NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").SetVisitorId(visitor).SetVisitorReady(false)
			},
			by: func(o, v uint32) uint32 { return o },
		},
		{
			name: "no visitor",
			build: func(h *harness) *Builder {
				return NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK")
			},
			by: func(o, v uint32) uint32 { return o },
		},
		{
			name:  "already in progress",
			build: func(h *harness) *Builder { return runningOmokBuilder(h, owner, visitor, 1) },
			by:    func(o, v uint32) uint32 { return o },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			seedRoom(t, h, tc.build(h))
			buf := message.NewBuffer()
			require.NoError(t, h.p.start(buf, uuid.New(), tc.by(owner, visitor)))
			assert.Empty(t, decodeEvents[minigame.StartedEventBody](t, buf, minigame.EventTypeStarted))
		})
	}
}

// --- MOVE_STONE --------------------------------------------------------------

func TestMoveStone_Valid(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8501)
	visitor := uint32(8502)
	// firstMover=1 -> owner (slot 0) moves first with color 1.
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.moveStone(buf, uuid.New(), owner, 7, 7))

	placed := decodeEvents[minigame.StonePlacedEventBody](t, buf, minigame.EventTypeStonePlaced)
	require.Len(t, placed, 1)
	assert.Equal(t, uint32(7), placed[0].Body.X)
	assert.Equal(t, byte(1), placed[0].Body.StoneType, "owner (first mover) plays color 1 (§G1)")

	r, _ := h.p.reg.Get(h.t, owner)
	assert.Equal(t, byte(1), r.CurrentTurn(), "turn flips to the visitor")
	require.Len(t, r.Moves(), 1)
	assert.Equal(t, byte(1), r.Board()[7*omok.BoardSize+7])
}

func TestMoveStone_OutOfTurnAndOccupied(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8601)
	visitor := uint32(8602)

	t.Run("out of turn", func(t *testing.T) {
		seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1)) // currentTurn = owner
		buf := message.NewBuffer()
		require.NoError(t, h.p.moveStone(buf, uuid.New(), visitor, 3, 3)) // visitor out of turn
		assert.Empty(t, buf.GetAll())
	})

	t.Run("occupied", func(t *testing.T) {
		h2 := newHarness(t)
		var board [omok.Cells]byte
		board[3*omok.BoardSize+3] = 2
		seedRoom(t, h2, runningOmokBuilder(h2, owner, visitor, 1).SetBoard(board))
		buf := message.NewBuffer()
		require.NoError(t, h2.p.moveStone(buf, uuid.New(), owner, 3, 3))
		assert.Empty(t, buf.GetAll())
	})
}

func TestMoveStone_WinningMoveEndsGame(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8701)
	visitor := uint32(8702)
	// Pre-seed 4 owner stones (color 1) in a row; owner plays the 5th.
	var board [omok.Cells]byte
	for x := 0; x < 4; x++ {
		board[x] = 1
	}
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1).SetBoard(board))

	buf := message.NewBuffer()
	require.NoError(t, h.p.moveStone(buf, uuid.New(), owner, 4, 0))

	require.Len(t, decodeEvents[minigame.StonePlacedEventBody](t, buf, minigame.EventTypeStonePlaced), 1)
	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, resultWin, ended[0].Body.ResultType)
	assert.Equal(t, byte(0), ended[0].Body.WinnerSlot)
	assert.Equal(t, uint32(1), ended[0].Body.OwnerRecord.Wins)
	assert.Equal(t, uint32(1), ended[0].Body.VisitorRecord.Losses)
	assert.Equal(t, int32(50), ended[0].Body.OwnerScore)
	assert.Equal(t, int32(15), ended[0].Body.VisitorScore)

	// Room reset for rematch, FirstMover advanced to the winner (owner -> 0).
	r, _ := h.p.reg.Get(h.t, owner)
	assert.False(t, r.InProgress())
	assert.Equal(t, byte(0), r.FirstMover())
	assert.Empty(t, r.Moves())
	assert.Equal(t, byte(0), r.Board()[0], "board wiped")

	// Record persisted to the DB (committed before emit).
	rec, err := record.GetOrZero(h.p.db, h.t.Id(), owner, record.GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), rec.Wins())
}

// --- FLIP_CARD ---------------------------------------------------------------

func TestFlipCard_FirstAndSecond(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8801)
	visitor := uint32(8802)
	deck := []uint32{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5}

	t.Run("first flip", func(t *testing.T) {
		seedRoom(t, h, runningMatchCardsBuilder(h, owner, visitor, 0, deck))
		buf := message.NewBuffer()
		require.NoError(t, h.p.flipCard(buf, uuid.New(), owner, true, 0))
		evs := decodeEvents[minigame.CardFlippedEventBody](t, buf, minigame.EventTypeCardFlipped)
		require.Len(t, evs, 1)
		assert.False(t, evs[0].Body.SecondFlip)
		assert.Equal(t, byte(0), evs[0].Body.Slot)
		r, _ := h.p.reg.Get(h.t, owner)
		assert.Equal(t, int16(0), r.FirstSlot())
	})

	t.Run("second flip match retains turn", func(t *testing.T) {
		h2 := newHarness(t)
		seedRoom(t, h2, runningMatchCardsBuilder(h2, owner, visitor, 0, deck).SetFirstSlot(0))
		buf := message.NewBuffer()
		require.NoError(t, h2.p.flipCard(buf, uuid.New(), owner, false, 1)) // deck[0]==deck[1]
		evs := decodeEvents[minigame.CardFlippedEventBody](t, buf, minigame.EventTypeCardFlipped)
		require.Len(t, evs, 1)
		assert.True(t, evs[0].Body.SecondFlip)
		assert.Equal(t, byte(2), evs[0].Body.ResultType, "owner match -> 2")
		r, _ := h2.p.reg.Get(h2.t, owner)
		assert.Equal(t, byte(1), r.OwnerPairs())
		assert.Equal(t, byte(0), r.CurrentTurn(), "match retains the turn")
	})

	t.Run("second flip mismatch passes turn", func(t *testing.T) {
		h3 := newHarness(t)
		seedRoom(t, h3, runningMatchCardsBuilder(h3, owner, visitor, 0, deck).SetFirstSlot(0))
		buf := message.NewBuffer()
		require.NoError(t, h3.p.flipCard(buf, uuid.New(), owner, false, 2)) // deck[0]!=deck[2]
		evs := decodeEvents[minigame.CardFlippedEventBody](t, buf, minigame.EventTypeCardFlipped)
		require.Len(t, evs, 1)
		assert.Equal(t, byte(0), evs[0].Body.ResultType, "owner mismatch -> 0")
		r, _ := h3.p.reg.Get(h3.t, owner)
		assert.Equal(t, byte(1), r.CurrentTurn(), "mismatch passes the turn")
	})
}

func TestFlipCard_Guards(t *testing.T) {
	h := newHarness(t)
	owner := uint32(8901)
	visitor := uint32(8902)
	deck := []uint32{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5}

	t.Run("out of turn", func(t *testing.T) {
		seedRoom(t, h, runningMatchCardsBuilder(h, owner, visitor, 0, deck)) // currentTurn owner
		buf := message.NewBuffer()
		require.NoError(t, h.p.flipCard(buf, uuid.New(), visitor, true, 0))
		assert.Empty(t, buf.GetAll())
	})

	t.Run("bad index", func(t *testing.T) {
		h2 := newHarness(t)
		seedRoom(t, h2, runningMatchCardsBuilder(h2, owner, visitor, 0, deck))
		buf := message.NewBuffer()
		require.NoError(t, h2.p.flipCard(buf, uuid.New(), owner, true, 99))
		assert.Empty(t, buf.GetAll())
	})
}

func TestFlipCard_LastPairWinAndTie(t *testing.T) {
	owner := uint32(9001)
	visitor := uint32(9002)
	deck := []uint32{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5} // pieceType 0 -> 6 pairs to win

	t.Run("owner takes last pair to win", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, runningMatchCardsBuilder(h, owner, visitor, 0, deck).
			SetOwnerPairs(5).SetVisitorPairs(0).SetFirstSlot(0).SetCurrentTurn(0))
		buf := message.NewBuffer()
		require.NoError(t, h.p.flipCard(buf, uuid.New(), owner, false, 1)) // match -> ownerPairs 6
		require.Len(t, decodeEvents[minigame.CardFlippedEventBody](t, buf, minigame.EventTypeCardFlipped), 1)
		ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
		require.Len(t, ended, 1)
		assert.Equal(t, resultWin, ended[0].Body.ResultType)
		assert.Equal(t, byte(0), ended[0].Body.WinnerSlot)
	})

	t.Run("equal pairs tie", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, runningMatchCardsBuilder(h, owner, visitor, 0, deck).
			SetOwnerPairs(3).SetVisitorPairs(2).SetFirstSlot(0).SetCurrentTurn(1))
		buf := message.NewBuffer()
		require.NoError(t, h.p.flipCard(buf, uuid.New(), visitor, false, 1)) // match -> visitorPairs 3 -> 3:3
		ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
		require.Len(t, ended, 1)
		assert.Equal(t, resultTie, ended[0].Body.ResultType)
	})
}

// --- SKIP --------------------------------------------------------------------

func TestSkip_TurnAndWho(t *testing.T) {
	owner := uint32(9101)
	visitor := uint32(9102)

	t.Run("owner skip -> Who 1", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1).SetCurrentTurn(0))
		buf := message.NewBuffer()
		require.NoError(t, h.p.skip(buf, uuid.New(), owner))
		evs := decodeEvents[minigame.SkippedEventBody](t, buf, minigame.EventTypeSkipped)
		require.Len(t, evs, 1)
		assert.Equal(t, byte(1), evs[0].Body.Who, "next mover = visitor")
		r, _ := h.p.reg.Get(h.t, owner)
		assert.Equal(t, byte(1), r.CurrentTurn())
	})

	t.Run("visitor skip -> Who 0", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1).SetCurrentTurn(1))
		buf := message.NewBuffer()
		require.NoError(t, h.p.skip(buf, uuid.New(), visitor))
		evs := decodeEvents[minigame.SkippedEventBody](t, buf, minigame.EventTypeSkipped)
		require.Len(t, evs, 1)
		assert.Equal(t, byte(0), evs[0].Body.Who, "next mover = owner")
	})
}

// --- TIE ---------------------------------------------------------------------

func TestTie_RequestGating(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9201)
	visitor := uint32(9202)

	t.Run("request when not denied", func(t *testing.T) {
		seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))
		buf := message.NewBuffer()
		require.NoError(t, h.p.requestTie(buf, uuid.New(), owner))
		evs := decodeEvents[minigame.EmptyEventBody](t, buf, minigame.EventTypeTieRequested)
		require.Len(t, evs, 1)
		assert.Equal(t, owner, evs[0].CharacterId)
	})

	t.Run("request when denied is dropped", func(t *testing.T) {
		h2 := newHarness(t)
		seedRoom(t, h2, runningOmokBuilder(h2, owner, visitor, 1).SetDeniedTie(0, true))
		buf := message.NewBuffer()
		require.NoError(t, h2.p.requestTie(buf, uuid.New(), owner))
		assert.Empty(t, buf.GetAll())
	})
}

func TestTie_AnswerAcceptEndsGame(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9301)
	visitor := uint32(9302)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.answerTie(buf, uuid.New(), visitor, true))

	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, resultTie, ended[0].Body.ResultType)
	assert.Equal(t, int32(10), ended[0].Body.OwnerScore)
	assert.Equal(t, int32(10), ended[0].Body.VisitorScore)
	assert.Equal(t, uint32(1), ended[0].Body.OwnerRecord.Ties)
	assert.Equal(t, uint32(1), ended[0].Body.VisitorRecord.Ties)
}

func TestTie_AnswerDeclineSetsDenyBit(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9401)
	visitor := uint32(9402)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.answerTie(buf, uuid.New(), visitor, false)) // visitor declines owner's tie

	evs := decodeEvents[minigame.AnswerEventBody](t, buf, minigame.EventTypeTieAnswered)
	require.Len(t, evs, 1)
	assert.False(t, evs[0].Body.Accept)

	// The requester (owner) is now denied and can no longer propose.
	r, _ := h.p.reg.Get(h.t, owner)
	assert.True(t, r.DeniedTie(0))
	buf2 := message.NewBuffer()
	require.NoError(t, h.p.requestTie(buf2, uuid.New(), owner))
	assert.Empty(t, buf2.GetAll())
}

// --- GIVE_UP -----------------------------------------------------------------

func TestGiveUp_Forfeit(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9501)
	visitor := uint32(9502)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.giveUp(buf, uuid.New(), owner)) // owner forfeits

	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, resultForfeit, ended[0].Body.ResultType)
	assert.Equal(t, byte(1), ended[0].Body.WinnerSlot, "visitor wins the forfeit")
	assert.Equal(t, int32(50), ended[0].Body.VisitorScore)
	assert.Equal(t, int32(-15), ended[0].Body.OwnerScore)

	r, _ := h.p.reg.Get(h.t, owner)
	assert.Equal(t, byte(1), r.OwnerForfeits())
}

// --- RETREAT -----------------------------------------------------------------

func TestRetreat_RequestAcceptDecline(t *testing.T) {
	owner := uint32(9601)
	visitor := uint32(9602)
	// owner just placed the tail stone (color 1); it's the visitor's turn.
	build := func(h *harness) *Builder {
		var board [omok.Cells]byte
		board[7*omok.BoardSize+7] = 1
		return runningOmokBuilder(h, owner, visitor, 1).
			SetCurrentTurn(1).
			SetBoard(board).
			SetMoves([]Move{{X: 7, Y: 7, Stone: 1}})
	}

	t.Run("request forwarded", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, build(h))
		buf := message.NewBuffer()
		require.NoError(t, h.p.requestRetreat(buf, uuid.New(), owner))
		require.Len(t, decodeEvents[minigame.EmptyEventBody](t, buf, minigame.EventTypeRetreatRequested), 1)
	})

	t.Run("accept pops stone and restores turn", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, build(h))
		buf := message.NewBuffer()
		require.NoError(t, h.p.answerRetreat(buf, uuid.New(), visitor, true))
		evs := decodeEvents[minigame.AnswerEventBody](t, buf, minigame.EventTypeRetreatAnswered)
		require.Len(t, evs, 1)
		assert.True(t, evs[0].Body.Accept)
		r, _ := h.p.reg.Get(h.t, owner)
		assert.Empty(t, r.Moves(), "tail stone popped")
		assert.Equal(t, byte(0), r.Board()[7*omok.BoardSize+7], "board cell cleared")
		assert.Equal(t, byte(0), r.CurrentTurn(), "turn restored to the requester (owner)")
	})

	t.Run("decline forwarded", func(t *testing.T) {
		h := newHarness(t)
		seedRoom(t, h, build(h))
		buf := message.NewBuffer()
		require.NoError(t, h.p.answerRetreat(buf, uuid.New(), visitor, false))
		evs := decodeEvents[minigame.AnswerEventBody](t, buf, minigame.EventTypeRetreatAnswered)
		require.Len(t, evs, 1)
		assert.False(t, evs[0].Body.Accept)
		r, _ := h.p.reg.Get(h.t, owner)
		require.Len(t, r.Moves(), 1, "decline leaves the board untouched")
	})
}

// --- MID-GAME LEAVE / EXPEL forfeit ------------------------------------------

func TestMidGameLeave_VisitorForfeits(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9701)
	visitor := uint32(9702)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.leave(buf, uuid.New(), visitor))

	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, resultForfeit, ended[0].Body.ResultType)
	assert.Equal(t, byte(0), ended[0].Body.WinnerSlot, "owner wins the forfeit")

	left := decodeEvents[minigame.LeftEventBody](t, buf, minigame.EventTypeLeft)
	require.Len(t, left, 1)
	assert.Equal(t, byte(4), left[0].Body.Status)

	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok, "room stays open after the visitor forfeits and leaves")
	assert.Equal(t, uint32(0), r.VisitorId())
	assert.False(t, r.InProgress())
}

func TestMidGameLeave_OwnerForfeitsClosesRoom(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9801)
	visitor := uint32(9802)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.leave(buf, uuid.New(), owner))

	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, byte(1), ended[0].Body.WinnerSlot, "visitor wins the forfeit")
	require.Len(t, decodeEvents[minigame.RoomClosedEventBody](t, buf, minigame.EventTypeRoomClosed), 1)

	_, ok := h.p.reg.Get(h.t, owner)
	assert.False(t, ok, "owner leaving closes the room after the forfeit")
}

func TestMidGameExpel_VisitorForfeits(t *testing.T) {
	h := newHarness(t)
	owner := uint32(9901)
	visitor := uint32(9902)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.expel(buf, uuid.New(), owner))

	ended := decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded)
	require.Len(t, ended, 1)
	assert.Equal(t, byte(0), ended[0].Body.WinnerSlot)
	left := decodeEvents[minigame.LeftEventBody](t, buf, minigame.EventTypeLeft)
	require.Len(t, left, 1)
	assert.Equal(t, byte(5), left[0].Body.Status, "expelled")
}

// --- EXIT_AFTER_GAME ---------------------------------------------------------

func TestExitAfter_OwnerClosesRoomAfterResult(t *testing.T) {
	h := newHarness(t)
	owner := uint32(10001)
	visitor := uint32(10002)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1).SetExitAfter(0, true))

	buf := message.NewBuffer()
	require.NoError(t, h.p.giveUp(buf, uuid.New(), visitor)) // visitor forfeits, owner wins

	require.Len(t, decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded), 1)
	require.Len(t, decodeEvents[minigame.RoomClosedEventBody](t, buf, minigame.EventTypeRoomClosed), 1)
	_, ok := h.p.reg.Get(h.t, owner)
	assert.False(t, ok, "owner's exit-after closes the room after the game")
}

func TestExitAfter_VisitorLeavesAfterResult(t *testing.T) {
	h := newHarness(t)
	owner := uint32(10101)
	visitor := uint32(10102)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1).SetExitAfter(1, true))

	buf := message.NewBuffer()
	require.NoError(t, h.p.giveUp(buf, uuid.New(), owner)) // owner forfeits, visitor wins

	require.Len(t, decodeEvents[minigame.GameEndedEventBody](t, buf, minigame.EventTypeGameEnded), 1)
	left := decodeEvents[minigame.LeftEventBody](t, buf, minigame.EventTypeLeft)
	require.Len(t, left, 1)
	assert.Equal(t, byte(4), left[0].Body.Status)
	r, ok := h.p.reg.Get(h.t, owner)
	require.True(t, ok, "room stays open (owner remains)")
	assert.Equal(t, uint32(0), r.VisitorId())
}

func TestExitAfter_Cancel(t *testing.T) {
	h := newHarness(t)
	owner := uint32(10201)
	visitor := uint32(10202)
	seedRoom(t, h, runningOmokBuilder(h, owner, visitor, 1))

	buf := message.NewBuffer()
	require.NoError(t, h.p.setExitAfter(buf, uuid.New(), owner, true))
	r, _ := h.p.reg.Get(h.t, owner)
	assert.True(t, r.ExitAfter(0))

	require.NoError(t, h.p.setExitAfter(buf, uuid.New(), owner, false))
	r, _ = h.p.reg.Get(h.t, owner)
	assert.False(t, r.ExitAfter(0))
}

// --- scoring matrix (pure transition) ----------------------------------------

func TestResolvedRoom_Scoring(t *testing.T) {
	now := time.Unix(1_600_000_000, 0).UTC()
	base := func() Room {
		return NewBuilder(RoomTypeOmok, 1, field.NewBuilder(1, 1, 100000).Build()).
			SetVisitorId(2).SetInProgress(true).Build()
	}

	t.Run("owner win", func(t *testing.T) {
		r := resolvedRoom(base(), resultWin, 0, now)
		assert.Equal(t, int32(50), r.OwnerScore())
		assert.Equal(t, int32(15), r.VisitorScore())
		assert.Equal(t, byte(0), r.FirstMover())
		assert.False(t, r.InProgress())
	})

	t.Run("visitor win", func(t *testing.T) {
		r := resolvedRoom(base(), resultWin, 1, now)
		assert.Equal(t, int32(50), r.VisitorScore())
		assert.Equal(t, int32(15), r.OwnerScore())
		assert.Equal(t, byte(1), r.FirstMover())
	})

	t.Run("forfeit awards winner and penalizes loser", func(t *testing.T) {
		r := resolvedRoom(base(), resultForfeit, 1, now) // owner forfeits
		assert.Equal(t, int32(50), r.VisitorScore())
		assert.Equal(t, int32(-15), r.OwnerScore())
		assert.Equal(t, byte(1), r.OwnerForfeits())
	})

	t.Run("forfeit farm guard suppresses the +50", func(t *testing.T) {
		farmed := Clone(base()).SetOwnerForfeits(forfeitFarmThreshold).Build()
		r := resolvedRoom(farmed, resultForfeit, 1, now) // owner forfeits again
		assert.Equal(t, int32(0), r.VisitorScore(), "winner's +50 suppressed once loser has >= 4 forfeits")
		assert.Equal(t, int32(-15), r.OwnerScore())
		assert.Equal(t, byte(5), r.OwnerForfeits())
	})

	t.Run("tie outside cooldown awards both", func(t *testing.T) {
		r := resolvedRoom(base(), resultTie, 0, now)
		assert.Equal(t, int32(10), r.OwnerScore())
		assert.Equal(t, int32(10), r.VisitorScore())
		assert.Equal(t, now.Add(tieCooldown), r.TieCooldownUntil())
		assert.Equal(t, byte(1), r.FirstMover(), "tie leaves FirstMover unchanged")
	})

	t.Run("tie inside cooldown awards nothing", func(t *testing.T) {
		cooled := Clone(base()).SetTieCooldownUntil(now.Add(time.Minute)).Build()
		r := resolvedRoom(cooled, resultTie, 0, now)
		assert.Equal(t, int32(0), r.OwnerScore())
		assert.Equal(t, int32(0), r.VisitorScore())
	})
}

// TestEndGame_Idempotent guards the single-resolution invariant: a room that is
// not InProgress must not resolve a second time (Cosmic minigameMatchFinish).
func TestEndGame_Idempotent(t *testing.T) {
	h := newHarness(t)
	owner := uint32(10301)
	visitor := uint32(10302)
	// Room already resolved (not in progress).
	seedRoom(t, h, NewBuilder(RoomTypeOmok, owner, h.f).SetGameType("OMOK").
		SetVisitorId(visitor).SetInProgress(false))

	buf := message.NewBuffer()
	require.NoError(t, h.p.endGame(buf, uuid.New(), owner, resultWin, 0))
	assert.Empty(t, buf.GetAll(), "endGame is a no-op when the game is not in progress")
}
