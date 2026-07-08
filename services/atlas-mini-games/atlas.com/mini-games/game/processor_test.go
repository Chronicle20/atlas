package game

import (
	"atlas-mini-games/kafka/message"
	"atlas-mini-games/kafka/message/minigame"
	"context"
	"encoding/json"
	"testing"

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
	p   *ProcessorImpl
	t   tenant.Model
	f   field.Model
	cp  fakeCharacter
	mp  fakeMap
	ip  *fakeInventory
	chp fakeChalkboard
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
	}
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
