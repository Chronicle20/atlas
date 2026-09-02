package character

import (
	"atlas-trades/escrow"
	characterKafka "atlas-trades/kafka/message/character"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/trade"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// harness is one test's wiring: the tenant its rooms live under, the context the
// handlers are called with, and the database whose outbox rows ARE the published
// batch. Commands publish through the transactional outbox, so asserting on those
// rows exercises the real path rather than a hand-built buffer.
type harness struct {
	t   *testing.T
	tm  tenant.Model
	ctx context.Context
	db  *gorm.DB
	p   trade.Processor
}

// logger is silenced: the teardown handlers log at Debug and Error, and an Error
// is an expected outcome of one of the tests below.
func logger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// newHarness derives the tenant from the test name, so tests sharing the
// process-wide trade.Registry singleton cannot see each other's rooms.
func newHarness(t *testing.T) *harness {
	t.Helper()
	tm, err := tenant.Create(uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name())), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	db := databasetest.NewInMemoryTenantDB(t, outbox.Migration, escrow.Migration)
	return &harness{t: t, tm: tm, ctx: ctx, db: db, p: trade.NewProcessor(logger(), ctx, db)}
}

// seatRoom registers a paired room owned by 100 with 200 as its visitor, in the
// given state. Nothing is staged, so no teardown path needs the inventory service.
//
// The Room is built with the exported trade.Builder — this project does not use
// test-only constructors — and installed directly in the registry rather than
// through the processor, because CreateRoom/EnterRoom run the whole validation
// ladder against REST services these tests do not stand up.
func (h *harness) seatRoom(state trade.State) trade.Room {
	h.t.Helper()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	room := trade.NewBuilder(miniroom.Trade, 100, "Owner", f).
		SetVisitor(200, "Guest").
		SetState(state).
		Build()
	if err := trade.GetRegistry().Create(h.tm, room); err != nil {
		h.t.Fatalf("seat room: %v", err)
	}
	h.t.Cleanup(func() { trade.GetRegistry().Remove(h.tm, room.Id()) })
	return room
}

// statusEvents decodes every trade status event of the given type that reached
// the outbox, in publication order. Topic tokens resolve to themselves in tests,
// because no topic env var is set.
func (h *harness) statusEvents(eventType string) []trademsg.StatusEvent[trademsg.CancelledEventBody] {
	h.t.Helper()
	var rows []outbox.Entity
	if err := h.db.Order("id asc").Find(&rows).Error; err != nil {
		h.t.Fatalf("read outbox: %v", err)
	}
	out := make([]trademsg.StatusEvent[trademsg.CancelledEventBody], 0)
	for _, r := range rows {
		if r.Topic != string(trademsg.EnvEventTopicStatus) {
			continue
		}
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(r.MessageValue, &probe); err != nil {
			h.t.Fatalf("probe status event: %v", err)
		}
		if probe.Type != eventType {
			continue
		}
		var ev trademsg.StatusEvent[trademsg.CancelledEventBody]
		if err := json.Unmarshal(r.MessageValue, &ev); err != nil {
			h.t.Fatalf("decode %s: %v", eventType, err)
		}
		out = append(out, ev)
	}
	return out
}

// assertCancelledWithReason requires exactly one CANCELLED event carrying the
// given leaveReason KEY.
func (h *harness) assertCancelledWithReason(reason string) {
	h.t.Helper()
	evs := h.statusEvents(trademsg.StatusTypeCancelled)
	if len(evs) != 1 {
		h.t.Fatalf("CANCELLED events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Reason != reason {
		h.t.Errorf("cancel reason: got %s, want %s", evs[0].Body.Reason, reason)
	}
}

// assertNoEventOfType requires that no status event of the given type reached the
// outbox.
func (h *harness) assertNoEventOfType(eventType string) {
	h.t.Helper()
	if evs := h.statusEvents(eventType); len(evs) != 0 {
		h.t.Errorf("%s events: got %d, want 0", eventType, len(evs))
	}
}

func logoutEvent(characterId uint32) characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody] {
	return characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(1),
		CharacterId:   characterId,
		Type:          characterKafka.EventCharacterStatusTypeLogout,
		Body:          characterKafka.StatusEventLogoutBody{ChannelId: channel.Id(1), MapId: _map.Id(100000000)},
	}
}

func mapChangedEvent(characterId uint32) characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody] {
	return characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(1),
		CharacterId:   characterId,
		Type:          characterKafka.EventCharacterStatusTypeMapChanged,
		Body: characterKafka.StatusEventMapChangedBody{
			ChannelId:   channel.Id(1),
			OldMapId:    _map.Id(100000000),
			TargetMapId: _map.Id(104000000),
		},
	}
}

func channelChangedEvent(characterId uint32) characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody] {
	return characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(1),
		CharacterId:   characterId,
		Type:          characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    channel.Id(2),
			OldChannelId: channel.Id(1),
			MapId:        _map.Id(100000000),
		},
	}
}

// TestLogoutTearsDownWithCancelledReason pins design §3.3: a disconnect ends the
// room and tells the survivor TRADE_CANCELLED. Escrow recovery does not depend on
// the disconnecting client being reachable (FR-6.4) — under the reserve-at-staging
// model there is nothing to recover, the reservations simply drop.
func TestLogoutTearsDownWithCancelledReason(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	handleStatusEventLogout(h.db)(logger(), h.ctx, logoutEvent(100))

	h.assertCancelledWithReason(trade.ReasonTradeCancelled)
	if _, ok := h.p.RoomForCharacter(charconst.Id(200)); ok {
		t.Error("survivor still in a room after the counterparty logged out")
	}
}

// TestMapChangeTearsDownWithDifferentMapReason pins design §3.3's row: a map
// change is TRADE_DIFFERENT_MAP, not TRADE_CANCELLED — the client has a distinct
// string for it.
func TestMapChangeTearsDownWithDifferentMapReason(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	handleStatusEventMapChanged(h.db)(logger(), h.ctx, mapChangedEvent(100))

	h.assertCancelledWithReason(trade.ReasonTradeDifferentMap)
	if _, ok := h.p.RoomForCharacter(charconst.Id(200)); ok {
		t.Error("survivor still in a room after the counterparty changed maps")
	}
}

// TestChannelChangeTearsDownWithDifferentMapReason pins the same row for a
// channel change, which emits neither LOGOUT nor MAP_CHANGED — without this arm
// the member index keeps the character bound to a dead room forever.
func TestChannelChangeTearsDownWithDifferentMapReason(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	handleStatusEventChannelChanged(h.db)(logger(), h.ctx, channelChangedEvent(100))

	h.assertCancelledWithReason(trade.ReasonTradeDifferentMap)
	if _, ok := h.p.RoomForCharacter(charconst.Id(200)); ok {
		t.Error("survivor still in a room after the counterparty changed channels")
	}
}

// TestTeardownIsIgnoredWhileSettling pins FR-6.5 at the consumer boundary too: a
// logout that lands while the settlement saga is in flight must neither announce
// a cancellation nor remove the room the saga's terminal status has to find.
func TestTeardownIsIgnoredWhileSettling(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateSettling)

	handleStatusEventLogout(h.db)(logger(), h.ctx, logoutEvent(100))

	h.assertNoEventOfType(trademsg.StatusTypeCancelled)
	if _, ok := h.p.RoomForCharacter(charconst.Id(100)); !ok {
		t.Error("a logout tore down a settling room")
	}
}

// TestTeardownOfACharacterWithoutARoomIsSilent pins that the consumers can be
// driven by every logout in the world: the overwhelming majority of characters
// are not trading. A logout for a character with no room must produce no event
// AND must not reach anyone else's room — this consumer sees every logout on the
// tenant, so a teardown keyed loosely would end an unrelated pair's trade.
func TestTeardownOfACharacterWithoutARoomIsSilent(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	handleStatusEventLogout(h.db)(logger(), h.ctx, logoutEvent(999))

	h.assertNoEventOfType(trademsg.StatusTypeCancelled)
	if _, ok := h.p.RoomForCharacter(charconst.Id(100)); !ok {
		t.Error("an unrelated character's logout tore down a live room")
	}
}

// TestNonTeardownTypesAreIgnored pins the type discrimination. All three handlers
// are subscribed to the SAME topic, so each receives every character-status
// message; a LOGIN body decodes cleanly into the logout body's shape, and a
// handler that did not check `type` first would tear down the room of a character
// who just logged in.
func TestNonTeardownTypesAreIgnored(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	login := logoutEvent(100)
	login.Type = characterKafka.EventCharacterStatusTypeLogin
	handleStatusEventLogout(h.db)(logger(), h.ctx, login)

	notAMapChange := mapChangedEvent(100)
	notAMapChange.Type = characterKafka.EventCharacterStatusTypeLogout
	handleStatusEventMapChanged(h.db)(logger(), h.ctx, notAMapChange)

	notAChannelChange := channelChangedEvent(100)
	notAChannelChange.Type = characterKafka.EventCharacterStatusTypeMapChanged
	handleStatusEventChannelChanged(h.db)(logger(), h.ctx, notAChannelChange)

	h.assertNoEventOfType(trademsg.StatusTypeCancelled)
	if _, ok := h.p.RoomForCharacter(charconst.Id(100)); !ok {
		t.Error("a non-teardown character status event tore the room down")
	}
}
