package trade

import (
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

// harness is one test's wiring, mirroring the teardown consumer's
// (kafka/consumer/character/consumer_test.go): commands publish through the
// transactional outbox, so the rows in outbox_entries ARE the published batch.
type harness struct {
	t   *testing.T
	tm  tenant.Model
	ctx context.Context
	db  *gorm.DB
}

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
	db := databasetest.NewInMemoryTenantDB(t, outbox.Migration)
	return &harness{t: t, tm: tm, ctx: ctx, db: db}
}

// seatRoom registers an OPEN paired room owned by 100 with 200 as its visitor.
// Nothing is staged, so no path here needs the inventory service. The Room is
// built with the exported trade.Builder and installed directly in the registry,
// because the create/enter ladders need REST services these tests do not stand
// up.
func (h *harness) seatRoom() trade.Room {
	h.t.Helper()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	room := trade.NewBuilder(miniroom.Trade, 100, "Owner", f).
		SetVisitor(200, "Guest").
		SetState(trade.StateOpen).
		Build()
	if err := trade.GetRegistry().Create(h.tm, room); err != nil {
		h.t.Fatalf("seat room: %v", err)
	}
	h.t.Cleanup(func() { trade.GetRegistry().Remove(h.tm, room.Id()) })
	return room
}

// rawEvents returns every trade status event of the given type that reached the
// outbox, undecoded. Topic tokens resolve to themselves in tests, because no
// topic env var is set.
func (h *harness) rawEvents(eventType string) [][]byte {
	h.t.Helper()
	var rows []outbox.Entity
	if err := h.db.Order("id asc").Find(&rows).Error; err != nil {
		h.t.Fatalf("read outbox: %v", err)
	}
	out := make([][]byte, 0)
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
		if probe.Type == eventType {
			out = append(out, r.MessageValue)
		}
	}
	return out
}

func decodeEvents[E any](t *testing.T, raws [][]byte) []trademsg.StatusEvent[E] {
	t.Helper()
	out := make([]trademsg.StatusEvent[E], 0, len(raws))
	for _, raw := range raws {
		var ev trademsg.StatusEvent[E]
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("decode: %v", err)
		}
		out = append(out, ev)
	}
	return out
}

func command[E any](commandType string, characterId charconst.Id, body E) trademsg.Command[E] {
	return trademsg.Command[E]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(1),
		MapId:         _map.Id(100000000),
		CharacterId:   characterId,
		Type:          commandType,
		Body:          body,
	}
}

// --- CANCEL -------------------------------------------------------------------

// TestCancelTearsTheRoomDown pins the arm that closes the ONLY teardown gap a
// logged-in player can reach: closing the trade dialog produces neither LOGOUT
// nor MAP_CHANGED nor CHANNEL_CHANGED, so without this handler the room would
// survive with its reservations being refreshed and the counterparty's dialog
// left open.
func TestCancelTearsTheRoomDown(t *testing.T) {
	h := newHarness(t)
	h.seatRoom()

	handleCancel(h.db)(logger(), h.ctx, command(trademsg.CommandTypeCancel, 100, trademsg.CancelCommandBody{}))

	evs := decodeEvents[trademsg.CancelledEventBody](t, h.rawEvents(trademsg.StatusTypeCancelled))
	if len(evs) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Reason != trade.ReasonTradeCancelled {
		t.Errorf("reason: got %s, want %s", evs[0].Body.Reason, trade.ReasonTradeCancelled)
	}
	if _, ok := trade.NewProcessor(logger(), h.ctx, h.db).RoomForCharacter(200); ok {
		t.Error("the survivor is still in a room after the counterparty cancelled")
	}
}

// TestCancelIgnoresAnotherCommandType pins the shared-topic guard: every handler
// registered on COMMAND_TOPIC_TRADE sees every command, so one that skipped the
// Type check would tear a room down on an unrelated command's body.
func TestCancelIgnoresAnotherCommandType(t *testing.T) {
	h := newHarness(t)
	h.seatRoom()

	handleCancel(h.db)(logger(), h.ctx, command(trademsg.CommandTypeConfirm, 100, trademsg.CancelCommandBody{}))

	if raws := h.rawEvents(trademsg.StatusTypeCancelled); len(raws) != 0 {
		t.Errorf("CANCELLED events: got %d, want 0", len(raws))
	}
	if _, ok := trade.NewProcessor(logger(), h.ctx, h.db).RoomForCharacter(100); !ok {
		t.Error("a CONFIRM command tore the room down through the cancel arm")
	}
}

// TestCancelFromACharacterWithNoRoomIsSilent pins that the unconditional EXIT
// fan-out in atlas-channel is safe: most EXITs belong to a mini-game or a shop.
func TestCancelFromACharacterWithNoRoomIsSilent(t *testing.T) {
	h := newHarness(t)

	handleCancel(h.db)(logger(), h.ctx, command(trademsg.CommandTypeCancel, 999, trademsg.CancelCommandBody{}))

	if raws := h.rawEvents(trademsg.StatusTypeCancelled); len(raws) != 0 {
		t.Errorf("CANCELLED events: got %d, want 0", len(raws))
	}
}

// --- CHAT ---------------------------------------------------------------------

// TestChatRelaysTheSpeakersLineWithTheirPosition pins that a trade-room chat
// line reaches the room, tagged with the side that spoke so atlas-channel can
// render the name prefix.
func TestChatRelaysTheSpeakersLineWithTheirPosition(t *testing.T) {
	h := newHarness(t)
	room := h.seatRoom()

	handleChat(h.db)(logger(), h.ctx, command(trademsg.CommandTypeChat, 200, trademsg.ChatCommandBody{Message: "deal?"}))

	evs := decodeEvents[trademsg.ChatEventBody](t, h.rawEvents(trademsg.StatusTypeChat))
	if len(evs) != 1 {
		t.Fatalf("CHAT events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Message != "deal?" {
		t.Errorf("message: got %q, want %q", evs[0].Body.Message, "deal?")
	}
	if evs[0].Body.Position != 1 {
		t.Errorf("position: got %d, want 1 (the visitor spoke)", evs[0].Body.Position)
	}
	if evs[0].CharacterId != 200 {
		t.Errorf("characterId: got %d, want 200", evs[0].CharacterId)
	}
	// Both sides ride the envelope so the channel can address the room without
	// a lookup.
	if evs[0].OwnerId != 100 || evs[0].VisitorId != 200 {
		t.Errorf("participants: got owner %d visitor %d, want 100/200", evs[0].OwnerId, evs[0].VisitorId)
	}
	if evs[0].RoomId != room.Id() {
		t.Errorf("roomId: got %s, want %s", evs[0].RoomId, room.Id())
	}
}

// TestChatIgnoresAnotherCommandType pins the shared-topic guard for the chat arm.
func TestChatIgnoresAnotherCommandType(t *testing.T) {
	h := newHarness(t)
	h.seatRoom()

	handleChat(h.db)(logger(), h.ctx, command(trademsg.CommandTypeCancel, 200, trademsg.ChatCommandBody{Message: "deal?"}))

	if raws := h.rawEvents(trademsg.StatusTypeChat); len(raws) != 0 {
		t.Errorf("CHAT events: got %d, want 0", len(raws))
	}
}

// TestChatFromANonMemberIsSilent pins that the serverbound CHAT fan-out — which
// reaches every mini-room family — is dropped for a speaker who is not in a
// trade room, rather than addressing someone else's room.
func TestChatFromANonMemberIsSilent(t *testing.T) {
	h := newHarness(t)
	h.seatRoom()

	handleChat(h.db)(logger(), h.ctx, command(trademsg.CommandTypeChat, 999, trademsg.ChatCommandBody{Message: "deal?"}))

	if raws := h.rawEvents(trademsg.StatusTypeChat); len(raws) != 0 {
		t.Errorf("CHAT events: got %d, want 0", len(raws))
	}
}

// --- ENTER_ROOM ----------------------------------------------------------------

// pendingRoom registers a room owned by 100 whose outstanding invite names 200.
// The handle defaults to the owner's character id (design §2.3), which is what
// makes it guessable and the admission gate necessary.
func (h *harness) pendingRoom() trade.Room {
	h.t.Helper()
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	room := trade.NewBuilder(miniroom.Trade, 100, "Owner", f).
		SetState(trade.StatePendingInvite).
		SetInvited(200).
		Build()
	if err := trade.GetRegistry().Create(h.tm, room); err != nil {
		h.t.Fatalf("pending room: %v", err)
	}
	h.t.Cleanup(func() { trade.GetRegistry().Remove(h.tm, room.Id()) })
	return room
}

// TestEnterRoomRefusesACharacterTheInviteDidNotName is the hijack attempt. The
// handle is the owner's character id, so an attacker in the same map can name
// it exactly; only the invited character may be seated. The refusal is
// ROOM_CLOSED — indistinguishable from an unknown handle, so the arm cannot be
// used to probe for live trades.
func TestEnterRoomRefusesACharacterTheInviteDidNotName(t *testing.T) {
	h := newHarness(t)
	room := h.pendingRoom()

	handleEnterRoom(h.db)(logger(), h.ctx, command(trademsg.CommandTypeEnterRoom, 300, trademsg.EnterRoomCommandBody{
		Handle:   room.Handle(),
		RoomType: room.RoomType(),
	}))

	evs := decodeEvents[trademsg.ErrorEventBody](t, h.rawEvents(trademsg.StatusTypeError))
	if len(evs) != 1 {
		t.Fatalf("ERROR events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Code != "ROOM_CLOSED" {
		t.Errorf("error code: got %s, want ROOM_CLOSED", evs[0].Body.Code)
	}
	p := trade.NewProcessor(logger(), h.ctx, h.db)
	if _, ok := p.RoomForCharacter(300); ok {
		t.Error("an uninvited character was seated in the room")
	}
	// The invitee's seat is still free for them.
	cur, ok := p.RoomById(room.Id())
	if !ok {
		t.Fatal("the room was destroyed by a refused enter")
	}
	if cur.VisitorId() != 0 {
		t.Errorf("visitorId: got %d, want 0", cur.VisitorId())
	}
	if !cur.Admits(200) {
		t.Error("the invitee lost their ticket to a refused enter")
	}
}

// TestEnterRoomRefusesAMismatchedRoomType pins the second half of the gate: a
// cash-trade enter must not seat anyone in a plain trade room, even when the
// enterer is the invited character.
func TestEnterRoomRefusesAMismatchedRoomType(t *testing.T) {
	h := newHarness(t)
	room := h.pendingRoom()

	handleEnterRoom(h.db)(logger(), h.ctx, command(trademsg.CommandTypeEnterRoom, 200, trademsg.EnterRoomCommandBody{
		Handle:   room.Handle(),
		RoomType: miniroom.CashTrade,
	}))

	evs := decodeEvents[trademsg.ErrorEventBody](t, h.rawEvents(trademsg.StatusTypeError))
	if len(evs) != 1 {
		t.Fatalf("ERROR events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Code != "ROOM_CLOSED" {
		t.Errorf("error code: got %s, want ROOM_CLOSED", evs[0].Body.Code)
	}
	if cur, ok := trade.NewProcessor(logger(), h.ctx, h.db).RoomById(room.Id()); !ok || cur.VisitorId() != 0 {
		t.Error("a mismatched room type still seated the enterer")
	}
}

// TestEnterRoomIgnoresAnotherCommandType pins the shared-topic guard for the
// enter arm.
func TestEnterRoomIgnoresAnotherCommandType(t *testing.T) {
	h := newHarness(t)
	room := h.pendingRoom()

	handleEnterRoom(h.db)(logger(), h.ctx, command(trademsg.CommandTypeCancel, 300, trademsg.EnterRoomCommandBody{
		Handle:   room.Handle(),
		RoomType: room.RoomType(),
	}))

	if raws := h.rawEvents(trademsg.StatusTypeError); len(raws) != 0 {
		t.Errorf("ERROR events: got %d, want 0", len(raws))
	}
}
