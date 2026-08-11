package session

import (
	"atlas-trades/escrow"
	sessionKafka "atlas-trades/kafka/message/session"
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
// handler is called with, and the database whose outbox rows ARE the published
// batch.
type harness struct {
	t   *testing.T
	tm  tenant.Model
	ctx context.Context
	db  *gorm.DB
	p   trade.Processor
}

// logger is silenced: the handler logs at Debug and Error.
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

// seatRoom registers a paired room owned by 100 with 200 as its visitor. Nothing
// is staged, so no teardown path needs the inventory service.
func (h *harness) seatRoom(state trade.State) trade.Room {
	h.t.Helper()
	return h.seatRoomOwnedBy(100, state)
}

// seatRoomOwnedBy registers a paired room with the given owner and 200 as its
// visitor. It uses the exported trade.Builder rather than the processor's
// CreateRoom/EnterRoom, whose validation ladder reads REST services these tests
// do not stand up.
func (h *harness) seatRoomOwnedBy(ownerId charconst.Id, state trade.State) trade.Room {
	h.t.Helper()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	room := trade.NewBuilder(miniroom.Trade, ownerId, "Owner", f).
		SetVisitor(200, "Guest").
		SetState(state).
		Build()
	if err := trade.GetRegistry().Create(h.tm, room); err != nil {
		h.t.Fatalf("seat room: %v", err)
	}
	h.t.Cleanup(func() { trade.GetRegistry().Remove(h.tm, room.Id()) })
	return room
}

// cancelledEvents decodes every CANCELLED status event that reached the outbox.
func (h *harness) cancelledEvents() []trademsg.StatusEvent[trademsg.CancelledEventBody] {
	h.t.Helper()
	var rows []outbox.Entity
	if err := h.db.Order("id asc").Find(&rows).Error; err != nil {
		h.t.Fatalf("read outbox: %v", err)
	}
	out := make([]trademsg.StatusEvent[trademsg.CancelledEventBody], 0)
	for _, r := range rows {
		if r.Topic != trademsg.EnvEventTopicStatus {
			continue
		}
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(r.MessageValue, &probe); err != nil {
			h.t.Fatalf("probe status event: %v", err)
		}
		if probe.Type != trademsg.StatusTypeCancelled {
			continue
		}
		var ev trademsg.StatusEvent[trademsg.CancelledEventBody]
		if err := json.Unmarshal(r.MessageValue, &ev); err != nil {
			h.t.Fatalf("decode CANCELLED: %v", err)
		}
		out = append(out, ev)
	}
	return out
}

func destroyedEvent(characterId uint32) sessionKafka.StatusEvent {
	return sessionKafka.StatusEvent{
		SessionId:   uuid.New(),
		AccountId:   7,
		CharacterId: characterId,
		WorldId:     world.Id(1),
		ChannelId:   channel.Id(1),
		Issuer:      "CHANNEL",
		Type:        sessionKafka.EventSessionStatusTypeDestroyed,
	}
}

// TestSessionDestroyedTearsDownWithCancelledReason pins design §3.3's disconnect
// row on the path a vanished client actually takes: a socket that drops may never
// produce a LOGOUT, and the survivor must still be told TRADE_CANCELLED.
func TestSessionDestroyedTearsDownWithCancelledReason(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	handleStatusEventDestroyed(h.db)(logger(), h.ctx, destroyedEvent(100))

	evs := h.cancelledEvents()
	if len(evs) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Reason != trade.ReasonTradeCancelled {
		t.Errorf("cancel reason: got %s, want %s", evs[0].Body.Reason, trade.ReasonTradeCancelled)
	}
	if _, ok := h.p.RoomForCharacter(charconst.Id(200)); ok {
		t.Error("survivor still in a room after the counterparty's session was destroyed")
	}
}

// TestSessionDestroyedIsIgnoredWhileSettling pins FR-6.5 on this arm too.
func TestSessionDestroyedIsIgnoredWhileSettling(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateSettling)

	handleStatusEventDestroyed(h.db)(logger(), h.ctx, destroyedEvent(100))

	if evs := h.cancelledEvents(); len(evs) != 0 {
		t.Errorf("CANCELLED events: got %d, want 0", len(evs))
	}
	if _, ok := h.p.RoomForCharacter(charconst.Id(100)); !ok {
		t.Error("a destroyed session tore down a settling room")
	}
}

// TestSessionWithoutACharacterIsIgnored pins the character-id-0 guard. The login
// server destroys sessions that never selected a character, so a stream of
// characterId-0 DESTROYED events is normal traffic; 0 is the zero value, not a
// trader, and it must never be resolved through the member index.
//
// The room here is deliberately OWNED BY 0 — the only state in which the guard is
// observable from outside. Without it, one anonymous login-server disconnect would
// end that room.
func TestSessionWithoutACharacterIsIgnored(t *testing.T) {
	h := newHarness(t)
	h.seatRoomOwnedBy(0, trade.StateOpen)

	handleStatusEventDestroyed(h.db)(logger(), h.ctx, destroyedEvent(0))

	if evs := h.cancelledEvents(); len(evs) != 0 {
		t.Errorf("CANCELLED events: got %d, want 0", len(evs))
	}
	if _, ok := h.p.RoomForCharacter(charconst.Id(0)); !ok {
		t.Error("a session destroyed with no character tore a room down")
	}
}

// TestNonDestroyedTypesAreIgnored pins the type discrimination. The topic carries
// CREATED as well, whose envelope is the SAME struct — an unfiltered handler would
// tear down the room of a character who just connected.
func TestNonDestroyedTypesAreIgnored(t *testing.T) {
	h := newHarness(t)
	h.seatRoom(trade.StateOpen)

	created := destroyedEvent(100)
	created.Type = "CREATED"
	handleStatusEventDestroyed(h.db)(logger(), h.ctx, created)

	if evs := h.cancelledEvents(); len(evs) != 0 {
		t.Errorf("CANCELLED events: got %d, want 0", len(evs))
	}
	if _, ok := h.p.RoomForCharacter(charconst.Id(100)); !ok {
		t.Error("a CREATED session status event tore the room down")
	}
}
