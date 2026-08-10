package trade

import (
	"atlas-trades/compartment"
	"atlas-trades/configuration"
	characterdata "atlas-trades/data/character"
	"atlas-trades/kafka/message"
	invitemsg "atlas-trades/kafka/message/invite"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/ledger"
	sagaproducer "atlas-trades/saga"
	"atlas-trades/settlement"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	invitec "github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// --- fakes for the injected seams -------------------------------------------

// testCharacter is one row of the fake character service.
type testCharacter struct {
	Id    character.Id
	Name  string
	Hp    uint16
	Level byte
	Meso  uint32
}

type fakeCharacters struct {
	rows map[character.Id]testCharacter
	err  error
}

func (f *fakeCharacters) GetById(characterId character.Id) (characterdata.Model, error) {
	if f.err != nil {
		return characterdata.Model{}, f.err
	}
	row, ok := f.rows[characterId]
	if !ok {
		return characterdata.Model{}, errors.New("character not found")
	}
	rm := characterdata.RestModel{Id: row.Id, Name: row.Name, Hp: row.Hp, Level: row.Level, Meso: row.Meso}
	return characterdata.Extract(rm)
}

// testMap is one row of the fake map-data service.
type testMap struct {
	Id              _map.Id
	TradeDisallowed bool
}

type fakeMaps struct {
	disallowed map[_map.Id]bool
	err        error
}

func (f *fakeMaps) FieldLimit(mapId _map.Id) (uint32, error) {
	if f.err != nil {
		return 0, f.err
	}
	if f.disallowed[mapId] {
		// The mini-room bit; see data/map/field_limit.go for its provenance.
		return 0x80, nil
	}
	return 0, nil
}

type fakeLocations struct {
	fields map[character.Id]field.Model
	err    error
}

func (f *fakeLocations) FieldOf(characterId character.Id) (field.Model, error) {
	if f.err != nil {
		return field.Model{}, f.err
	}
	fm, ok := f.fields[characterId]
	if !ok {
		return field.Model{}, errors.New("no location")
	}
	return fm, nil
}

type fakeConfig struct {
	cfg configuration.Model
}

func (f fakeConfig) Get(_ logrus.FieldLogger, _ context.Context) configuration.Model { return f.cfg }

// --- harness -----------------------------------------------------------------

// emitted reads back everything a command published. Commands emit through the
// transactional outbox, so the rows in outbox_entries ARE the published batch —
// asserting on them exercises the real public path rather than a hand-built
// buffer.
type emitted struct {
	db *gorm.DB
}

// messages returns the values published to one topic token, in publication
// order. Tokens resolve to themselves in tests, because no topic env var is set
// (topic.EnvProvider falls back to the token).
func (e *emitted) messages(t *testing.T, token string) [][]byte {
	t.Helper()
	var rows []outbox.Entity
	if err := e.db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	var out [][]byte
	for _, r := range rows {
		if r.Topic == token {
			out = append(out, r.MessageValue)
		}
	}
	return out
}

// statusEvents decodes every trade status event of the given type.
func statusEvents[E any](t *testing.T, e *emitted, eventType string) []trademsg.StatusEvent[E] {
	t.Helper()
	var out []trademsg.StatusEvent[E]
	for _, raw := range e.messages(t, trademsg.EnvEventTopicStatus) {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatalf("probe status event: %v", err)
		}
		if probe.Type != eventType {
			continue
		}
		var ev trademsg.StatusEvent[E]
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("decode %s: %v", eventType, err)
		}
		out = append(out, ev)
	}
	return out
}

// assertErrorEvent requires exactly one ERROR event carrying the given
// enterError key.
func assertErrorEvent(t *testing.T, e *emitted, code string) {
	t.Helper()
	evs := statusEvents[trademsg.ErrorEventBody](t, e, trademsg.StatusTypeError)
	if len(evs) != 1 {
		t.Fatalf("ERROR events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Code != code {
		t.Errorf("error code: got %s, want %s", evs[0].Body.Code, code)
	}
}

// assertInviteRejected requires exactly one INVITE_REJECTED carrying the given
// inviteResult key.
func assertInviteRejected(t *testing.T, e *emitted, code string) {
	t.Helper()
	evs := statusEvents[trademsg.ErrorEventBody](t, e, trademsg.StatusTypeInviteRejected)
	if len(evs) != 1 {
		t.Fatalf("INVITE_REJECTED events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Code != code {
		t.Errorf("invite result: got %s, want %s", evs[0].Body.Code, code)
	}
}

// assertNoEventOfType requires that no status event of the given type was
// published.
func assertNoEventOfType(t *testing.T, e *emitted, eventType string) {
	t.Helper()
	if evs := statusEvents[json.RawMessage](t, e, eventType); len(evs) != 0 {
		t.Errorf("%s events: got %d, want 0", eventType, len(evs))
	}
}

// assertInviteCommand requires exactly one COMMAND_TOPIC_INVITE message and
// returns it.
func assertInviteCommand(t *testing.T, e *emitted) invitemsg.Command[invitemsg.CreateCommandBody] {
	t.Helper()
	raws := e.messages(t, invitemsg.EnvCommandTopic)
	if len(raws) != 1 {
		t.Fatalf("invite commands: got %d, want 1", len(raws))
	}
	var cmd invitemsg.Command[invitemsg.CreateCommandBody]
	if err := json.Unmarshal(raws[0], &cmd); err != nil {
		t.Fatalf("decode invite command: %v", err)
	}
	return cmd
}

// lifecycleTenant derives a tenant from the test name, so tests sharing the
// process-wide registry singleton cannot see each other's rooms.
func lifecycleTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name())), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

// newLifecycleProcessor builds a processor with every gate open: both test
// characters alive, level 30, standing in testField, on a map that permits
// trade, under the shipped default configuration.
func newLifecycleProcessor(t *testing.T, cfg configuration.Model, characters ...testCharacter) (*ProcessorImpl, *emitted) {
	t.Helper()

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	tm := lifecycleTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	db := databasetest.NewInMemoryTenantDB(t, outbox.Migration, ledger.Migration, settlement.Migration)

	rows := make(map[character.Id]testCharacter)
	locations := make(map[character.Id]field.Model)
	for _, c := range characters {
		rows[c.Id] = c
		locations[c.Id] = testField(t)
	}

	p := &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tm,
		reg:  GetRegistry(),
		cfg:  fakeConfig{cfg: cfg},
		cp:   &fakeCharacters{rows: rows},
		mp:   &fakeMaps{disallowed: make(map[_map.Id]bool)},
		locp: &fakeLocations{fields: locations},
		// The reservation producer only buffers messages, so the real one is
		// used rather than a fake: the assertions in the staging suite are
		// about the actual COMMAND_TOPIC_COMPARTMENT bytes.
		resp: compartment.NewProcessor(l, ctx),
		// Likewise the saga submitter: the settlement suite asserts on the
		// actual COMMAND_TOPIC_SAGA bytes, including the concrete payload type.
		sgp: sagaproducer.NewProcessor(l, ctx),
		// A PER-TEST deadline registry, not the process singleton. Tests share
		// the process, and an attestation deadline armed by one of them would
		// otherwise still be sleeping — holding a reference to that test's
		// in-memory database — long after it returned.
		timers: newAttestationTimers(),
	}
	t.Cleanup(p.timers.StopAll)
	return p, &emitted{db: db}
}

// defaultCharacters are the three ids the lifecycle tests trade between, all
// alive and above any default level gate.
func defaultCharacters() []testCharacter {
	return []testCharacter{
		{Id: 100, Name: "Owner", Hp: 100, Level: 30, Meso: 1_000_000},
		{Id: 200, Name: "Guest", Hp: 100, Level: 30, Meso: 1_000_000},
		{Id: 300, Name: "Third", Hp: 100, Level: 30, Meso: 1_000_000},
	}
}

func testProcessor(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	return newLifecycleProcessor(t, configuration.DefaultConfig(), defaultCharacters()...)
}

func testProcessorWithCharacter(t *testing.T, c testCharacter) (*ProcessorImpl, *emitted) {
	t.Helper()
	rows := defaultCharacters()
	for i := range rows {
		if rows[i].Id == c.Id {
			rows[i] = c
		}
	}
	return newLifecycleProcessor(t, configuration.DefaultConfig(), rows...)
}

func testProcessorWithMap(t *testing.T, m testMap) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testProcessor(t)
	p.mp = &fakeMaps{disallowed: map[_map.Id]bool{m.Id: m.TradeDisallowed}}
	return p, e
}

func testProcessorWithConfig(t *testing.T, cfg configuration.Model, c testCharacter) (*ProcessorImpl, *emitted) {
	t.Helper()
	rows := defaultCharacters()
	for i := range rows {
		if rows[i].Id == c.Id {
			rows[i] = c
		}
	}
	return newLifecycleProcessor(t, cfg, rows...)
}

// testPendingRoom is a room whose owner (100) has invited 200 and is waiting.
func testPendingRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	return p, e
}

// --- CREATE ------------------------------------------------------------------

// TestCreateRoomStartsSoloWithCapacityTwo pins FR-1.1: the room starts with
// exactly one occupant at position 0.
func TestCreateRoomStartsSoloWithCapacityTwo(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("room not registered")
	}
	if room.State() != StateOpenSolo {
		t.Errorf("state: got %s, want %s", room.State(), StateOpenSolo)
	}
	if len(room.Participants()) != 1 {
		t.Errorf("participants: got %d, want 1", len(room.Participants()))
	}
	if room.Handle() != 100 {
		t.Errorf("handle: got %d, want the owner's character id 100", room.Handle())
	}

	created := statusEvents[trademsg.RoomCreatedEventBody](t, e, trademsg.StatusTypeRoomCreated)
	if len(created) != 1 {
		t.Fatalf("ROOM_CREATED events: got %d, want 1", len(created))
	}
	if created[0].Body.Position != 0 {
		t.Errorf("position: got %d, want 0", created[0].Body.Position)
	}
	if created[0].OwnerId != 100 {
		t.Errorf("ownerId: got %d, want 100", created[0].OwnerId)
	}
}

// TestCreateRoomRejectsDeadCharacter pins FR-4.7: a dead character gets
// NOT_WHEN_DEAD, not a room.
func TestCreateRoomRejectsDeadCharacter(t *testing.T) {
	p, e := testProcessorWithCharacter(t, testCharacter{Id: 100, Name: "Owner", Hp: 0, Level: 30})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create must buffer an error event, not return: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a dead character got a room")
	}
	assertErrorEvent(t, e, errNotWhenDead)
}

// TestCreateRoomRejectsTradeDisallowedMap pins FR-4.6.
func TestCreateRoomRejectsTradeDisallowedMap(t *testing.T) {
	p, e := testProcessorWithMap(t, testMap{Id: testField(t).MapId(), TradeDisallowed: true})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a trade-disallowed map allowed a room")
	}
	assertErrorEvent(t, e, errTradeNotAllowed)
}

// TestCreateRoomRefusesWhenTheFieldLimitCannotBeRead pins design §7: a flag that
// cannot be read must not be treated as "tradeable".
func TestCreateRoomRefusesWhenTheFieldLimitCannotBeRead(t *testing.T) {
	p, e := testProcessor(t)
	p.mp = &fakeMaps{err: errors.New("atlas-data unreachable")}
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("an unreadable field limit still opened a room")
	}
	assertErrorEvent(t, e, errTradeNotAllowed)
}

// TestCreateRoomRejectsBelowMinLevel pins FR-4.5 against a tenant config with a
// non-default minimum.
func TestCreateRoomRejectsBelowMinLevel(t *testing.T) {
	p, e := testProcessorWithConfig(t, configuration.DefaultConfig().WithMinTradeLevel(20), testCharacter{Id: 100, Name: "Owner", Hp: 100, Level: 10})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a below-minimum-level character got a room")
	}
	assertErrorEvent(t, e, errUnable)
}

// TestCreateRoomAllowsExactlyTheMinimumLevel pins that the level gate is a
// minimum, not a strict greater-than.
func TestCreateRoomAllowsExactlyTheMinimumLevel(t *testing.T) {
	p, _ := testProcessorWithConfig(t, configuration.DefaultConfig().WithMinTradeLevel(20), testCharacter{Id: 100, Name: "Owner", Hp: 100, Level: 20})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a character at exactly the minimum level was refused a room")
	}
}

// TestCreateRoomRejectsSecondRoom pins FR-1.2's authoritative half.
func TestCreateRoomRejectsSecondRoom(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("second create: %v", err)
	}
	assertErrorEvent(t, e, errOtherRequests)
}

// --- INVITE ------------------------------------------------------------------

// TestInviteMovesRoomToPendingAndProducesTradeInvite pins FR-2.1: atlas-trades
// is the first production producer of invite.TypeTrade, and the referenceId is
// the room's uint32 handle (a uuid does not fit).
func TestInviteMovesRoomToPendingAndProducesTradeInvite(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if room.State() != StatePendingInvite {
		t.Errorf("state: got %s, want %s", room.State(), StatePendingInvite)
	}
	cmd := assertInviteCommand(t, e)
	if cmd.InviteType != invitec.TypeTrade {
		t.Errorf("inviteType: got %s, want TRADE", cmd.InviteType)
	}
	if cmd.Type != invitec.CommandTypeCreate {
		t.Errorf("commandType: got %s, want CREATE", cmd.Type)
	}
	if cmd.Body.ReferenceId != invitec.Id(room.Handle()) {
		t.Errorf("referenceId: got %d, want the room handle %d", cmd.Body.ReferenceId, room.Handle())
	}
	if cmd.Body.OriginatorId != 100 || cmd.Body.TargetId != 200 {
		t.Errorf("invite parties: got originator %d target %d, want 100/200", cmd.Body.OriginatorId, cmd.Body.TargetId)
	}

	sent := statusEvents[trademsg.InviteSentEventBody](t, e, trademsg.StatusTypeInviteSent)
	if len(sent) != 1 {
		t.Fatalf("INVITE_SENT events: got %d, want 1", len(sent))
	}
	if sent[0].Body.InviterName != "Owner" {
		t.Errorf("inviterName: got %q, want %q", sent[0].Body.InviterName, "Owner")
	}
}

// TestInviteRejectsUnknownTarget pins that a target atlas-character cannot
// resolve is refused with the client's "unable to find the character" result,
// and does NOT move the room to PENDING_INVITE.
func TestInviteRejectsUnknownTarget(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 999); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertInviteRejected(t, e, inviteResultCannotFind)
	assertNoEventOfType(t, e, trademsg.StatusTypeInviteSent)
	room, _ := p.RoomForCharacter(100)
	if room.State() != StateOpenSolo {
		t.Errorf("state: got %s, want the room to stay %s", room.State(), StateOpenSolo)
	}
}

// TestInviteRejectsTargetOnAnotherMap pins FR-2.2: an invite may only reach a
// character in the same field.
func TestInviteRejectsTargetOnAnotherMap(t *testing.T) {
	p, e := testProcessor(t)
	p.locp = &fakeLocations{fields: map[character.Id]field.Model{
		100: testField(t),
		200: field.NewBuilder(1, 1, 200000000).Build(),
	}}
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertInviteRejected(t, e, inviteResultCannotFind)
}

// TestInviteRejectsDeadTarget pins FR-4.7 on the target side.
func TestInviteRejectsDeadTarget(t *testing.T) {
	p, e := testProcessorWithCharacter(t, testCharacter{Id: 200, Name: "Guest", Hp: 0, Level: 30})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertInviteRejected(t, e, inviteResultBusy)
}

// TestInviteRejectsTargetAlreadyTrading pins FR-1.2 on the target side: the
// client's own message for this is "is doing something else right now".
func TestInviteRejectsTargetAlreadyTrading(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create owner room: %v", err)
	}
	if err := p.CreateRoom(uuid.New(), testField(t), 200, miniroom.Trade); err != nil {
		t.Fatalf("create target room: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertInviteRejected(t, e, inviteResultBusy)
}

// TestInviteRejectsSelf pins that a character cannot invite themselves — the
// registry would otherwise reject the seat much later, after the invite had
// already gone out.
func TestInviteRejectsSelf(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 100); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertInviteRejected(t, e, inviteResultCannotFind)
	if len(e.messages(t, invitemsg.EnvCommandTopic)) != 0 {
		t.Error("a self-invite still produced a COMMAND_TOPIC_INVITE message")
	}
}

// TestInviteRejectedWithoutARoom pins the rejected transition: INVITE from a
// character who never created a room is UNABLE, not a crash.
func TestInviteRejectedWithoutARoom(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertErrorEvent(t, e, errUnable)
	assertNoEventOfType(t, e, trademsg.StatusTypeInviteSent)
}

// TestInviteRejectedOnAPairedRoom pins the rejected transition OPEN -> INVITE:
// once both seats are taken, no further invite may be issued.
func TestInviteRejectedOnAPairedRoom(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 300); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertErrorEvent(t, e, errUnable)
	room, _ = p.RoomForCharacter(100)
	if room.State() != StateOpen {
		t.Errorf("state: got %s, want the room to stay %s", room.State(), StateOpen)
	}
}

// TestInviteRejectedForTheVisitor pins that only the owner invites: the visitor
// re-inviting would otherwise drag a paired room back to PENDING_INVITE.
func TestInviteRejectedForTheVisitor(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 200, 300); err != nil {
		t.Fatalf("invite: %v", err)
	}
	assertErrorEvent(t, e, errUnable)
}

// TestInviteMayBeReissuedWhilePending pins design §3.1: PENDING_INVITE is
// itself an invitable state, so the owner of a room whose invite has not been
// answered may re-issue it (to the same or another target) without the room
// being torn down first.
func TestInviteMayBeReissuedWhilePending(t *testing.T) {
	p, e := testPendingRoom(t)
	if err := p.Invite(uuid.New(), testField(t), 100, 300); err != nil {
		t.Fatalf("second invite: %v", err)
	}
	if got := len(e.messages(t, invitemsg.EnvCommandTopic)); got != 2 {
		t.Errorf("invite commands: got %d, want 2", got)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeError)
}

// --- DECLINE -----------------------------------------------------------------

// TestDeclineDestroysTheRoom pins FR-2.5 / design §3.1: a decline tears the
// pending room down (the reference client closes the inviter's dialog).
func TestDeclineDestroysTheRoom(t *testing.T) {
	p, e := testPendingRoom(t)
	if err := p.DeclineInvite(uuid.New(), 200, 100); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a declined invite")
	}
	cancelled := statusEvents[trademsg.CancelledEventBody](t, e, trademsg.StatusTypeCancelled)
	if len(cancelled) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(cancelled))
	}
	if cancelled[0].Body.Reason != ReasonTradeCancelled {
		t.Errorf("reason: got %s, want %s", cancelled[0].Body.Reason, ReasonTradeCancelled)
	}
}

// TestDeclineRetiresTheInviteInAtlasInvites pins that a client-originated
// decline sends a REJECT command. Without it the invite lingers, and because a
// room's handle defaults to the owner's character id, the owner's next room
// reuses the same referenceId and the invite registry's dedup hands back the
// stale invite instead of raising a fresh dialog.
func TestDeclineRetiresTheInviteInAtlasInvites(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.DeclineInvite(uuid.New(), 200, 100); err != nil {
		t.Fatalf("decline: %v", err)
	}
	raws := e.messages(t, invitemsg.EnvCommandTopic)
	if len(raws) != 2 {
		t.Fatalf("invite commands: got %d, want the CREATE plus a REJECT", len(raws))
	}
	var rej invitemsg.Command[invitemsg.RejectCommandBody]
	if err := json.Unmarshal(raws[1], &rej); err != nil {
		t.Fatalf("decode reject command: %v", err)
	}
	if rej.Type != invitec.CommandTypeReject {
		t.Errorf("commandType: got %s, want REJECT", rej.Type)
	}
	if rej.InviteType != invitec.TypeTrade {
		t.Errorf("inviteType: got %s, want TRADE", rej.InviteType)
	}
	if rej.Body.TargetId != 200 || rej.Body.OriginatorId != 100 {
		t.Errorf("reject parties: got target %d originator %d, want 200/100", rej.Body.TargetId, rej.Body.OriginatorId)
	}
	if rej.WorldId != room.Field().WorldId() {
		t.Errorf("worldId: got %d, want %d", rej.WorldId, room.Field().WorldId())
	}
}

// TestInviteRejectedDoesNotEchoARejectBack pins the other direction: when
// atlas-invites told US the invite was rejected or expired, it has already
// deleted it, so a reject command would only make it fail to find the invite.
func TestInviteRejectedDoesNotEchoARejectBack(t *testing.T) {
	p, e := testPendingRoom(t)
	if err := p.InviteRejected(uuid.New(), 200, 100); err != nil {
		t.Fatalf("invite rejected: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a rejected invite")
	}
	if got := len(e.messages(t, invitemsg.EnvCommandTopic)); got != 1 {
		t.Errorf("invite commands: got %d, want only the original CREATE", got)
	}
	cancelled := statusEvents[trademsg.CancelledEventBody](t, e, trademsg.StatusTypeCancelled)
	if len(cancelled) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(cancelled))
	}
}

// TestDeclineLeavesASoloRoomAlone pins the rejected transition: a decline
// arriving for a room that never issued an invite (or whose invite was already
// answered) must not destroy it.
func TestDeclineLeavesASoloRoomAlone(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.DeclineInvite(uuid.New(), 200, 100); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a decline destroyed a room that had issued no invite")
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
}

// TestDeclineForAnUnknownOriginatorIsANoOp pins that a late decline — the room
// is already gone — neither errors nor emits.
func TestDeclineForAnUnknownOriginatorIsANoOp(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.DeclineInvite(uuid.New(), 200, 100); err != nil {
		t.Fatalf("decline: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
}

// --- ENTER -------------------------------------------------------------------

// TestEnterRoomSeatsVisitorAtPositionOne pins FR-1.5 / FR-2.4.
func TestEnterRoomSeatsVisitorAtPositionOne(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	room, ok := p.RoomForCharacter(200)
	if !ok {
		t.Fatal("the visitor was not indexed into the room")
	}
	if room.State() != StateOpen {
		t.Errorf("state: got %s, want %s", room.State(), StateOpen)
	}
	pt, ok := room.ParticipantFor(200)
	if !ok || pt.Position() != 1 {
		t.Errorf("visitor position: got %+v", pt)
	}
	if pt.Name() != "Guest" {
		t.Errorf("visitor name: got %q, want %q", pt.Name(), "Guest")
	}

	entered := statusEvents[trademsg.ParticipantEnteredEventBody](t, e, trademsg.StatusTypeParticipantEntered)
	if len(entered) != 1 {
		t.Fatalf("PARTICIPANT_ENTERED events: got %d, want 1", len(entered))
	}
	if entered[0].Body.Position != 1 || entered[0].Body.CharacterId != 200 {
		t.Errorf("entered body: got %+v, want character 200 at position 1", entered[0].Body)
	}
	if entered[0].VisitorId != 200 {
		t.Errorf("visitorId: got %d, want 200", entered[0].VisitorId)
	}
}

// TestEnterRoomRefusesACharacterTheInviteDidNotName drives the hijack through
// the FULL ladder, with every REST gate open and the attacker standing in the
// same map as the room: the only thing stopping them is the admission ticket.
// The wire handle is the owner's character id (design §2.3), so possession of
// it proves nothing.
func TestEnterRoomRefusesACharacterTheInviteDidNotName(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.EnterRoom(uuid.New(), testField(t), 300, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}

	assertErrorEvent(t, e, errRoomClosed)
	if _, ok := p.RoomForCharacter(300); ok {
		t.Error("an uninvited character was seated in the room")
	}
	// And the character the invite DID name is still able to take their seat.
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("invitee enter: %v", err)
	}
	seated, ok := p.RoomForCharacter(200)
	if !ok {
		t.Fatal("the invited character was not seated after the refused hijack")
	}
	if seated.State() != StateOpen {
		t.Errorf("state: got %s, want %s", seated.State(), StateOpen)
	}
}

// TestEnterRoomRefusesAMismatchedRoomType pins the second half of the gate: a
// cash-trade enter must not seat anyone in a plain trade room, even when the
// enterer is the invited character.
func TestEnterRoomRefusesAMismatchedRoomType(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), miniroom.CashTrade); err != nil {
		t.Fatalf("enter: %v", err)
	}

	assertErrorEvent(t, e, errRoomClosed)
	if _, ok := p.RoomForCharacter(200); ok {
		t.Error("a mismatched room type still seated the enterer")
	}
}

// TestReInvitingMovesTheTicket pins that a re-issued invite after a decline
// hands the seat to the NEW target and takes it from the old one — the ticket
// is the room's current invite, not a growing allow-list.
func TestReInvitingMovesTheTicket(t *testing.T) {
	p, _ := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.Invite(uuid.New(), testField(t), 100, 300); err != nil {
		t.Fatalf("re-invite: %v", err)
	}

	cur, ok := p.RoomById(room.Id())
	if !ok {
		t.Fatal("the room is gone after a re-invite")
	}
	if !cur.Admits(300) {
		t.Error("the re-invited character was not admitted")
	}
	if cur.Admits(200) {
		t.Error("the previously invited character kept their ticket")
	}
}

// TestEnterRoomRejectsAThirdCharacter pins that a paired room is closed. The
// third character is refused by the admission gate before the paired check is
// even reached — seating spends the invite ticket — so the code is ROOM_CLOSED,
// the same answer an unknown handle gets.
func TestEnterRoomRejectsAThirdCharacter(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("first enter: %v", err)
	}
	if err := p.EnterRoom(uuid.New(), testField(t), 300, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("second enter: %v", err)
	}
	assertErrorEvent(t, e, errRoomClosed)
	if _, ok := p.RoomForCharacter(300); ok {
		t.Error("a third character was seated in a paired room")
	}
}

// TestEnterRoomRejectsASoloRoom pins the rejected transition OPEN_SOLO ->
// ENTER: a room whose owner never invited anyone admits nobody. With no invite
// there is no ticket, so the refusal is ROOM_CLOSED.
func TestEnterRoomRejectsASoloRoom(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errRoomClosed)
}

// TestEnterRoomRejectsAnUnknownHandle pins that an accept for a room that is
// already gone reports ROOM_CLOSED rather than silently dropping.
func TestEnterRoomRejectsAnUnknownHandle(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, 4242, miniroom.Trade); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errRoomClosed)
}

// TestEnterRoomRejectsDeadCharacter pins FR-4.7 at accept time: the invite may
// have been sent while the acceptor was still alive.
func TestEnterRoomRejectsDeadCharacter(t *testing.T) {
	p, e := testPendingRoom(t)
	p.cp = &fakeCharacters{rows: map[character.Id]testCharacter{
		100: {Id: 100, Name: "Owner", Hp: 100, Level: 30},
		200: {Id: 200, Name: "Guest", Hp: 0, Level: 30},
	}}
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errNotWhenDead)
	if _, ok := p.RoomForCharacter(200); ok {
		t.Error("a dead character was seated")
	}
}

// TestEnterRoomRejectsBelowMinLevel pins FR-4.5 at accept time.
func TestEnterRoomRejectsBelowMinLevel(t *testing.T) {
	p, e := newLifecycleProcessor(t, configuration.DefaultConfig().WithMinTradeLevel(20),
		testCharacter{Id: 100, Name: "Owner", Hp: 100, Level: 30},
		testCharacter{Id: 200, Name: "Guest", Hp: 100, Level: 10},
	)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errUnable)
}

// TestEnterRoomRejectsADifferentField pins that a character who left the room's
// map between invite and accept is not seated into a room they cannot see.
//
// It passes room.Field() as f — exactly what the production invite-accept path
// does (kafka/consumer/invite/consumer.go) — and moves the enterer via the
// LOCATION service instead. Comparing f to the room would be a tautology, so a
// check that trusted f would pass this test while seating the character.
func TestEnterRoomRejectsADifferentField(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	elsewhere := field.NewBuilder(1, 1, 200000000).Build()
	p.locp = &fakeLocations{fields: map[character.Id]field.Model{
		100: room.Field(),
		200: elsewhere,
	}}
	if err := p.EnterRoom(uuid.New(), room.Field(), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errNotSameMap)
	if _, ok := p.RoomForCharacter(200); ok {
		t.Error("a character standing on another map was seated")
	}
}

// TestEnterRoomRefusesWhenTheEnterersLocationCannotBeRead pins that an
// unreadable location is a refusal, not a default-seat.
func TestEnterRoomRefusesWhenTheEnterersLocationCannotBeRead(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	p.locp = &fakeLocations{err: errors.New("atlas-maps unreachable")}
	if err := p.EnterRoom(uuid.New(), room.Field(), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errUnable)
	if _, ok := p.RoomForCharacter(200); ok {
		t.Error("an unreadable location still seated the character")
	}
}

// TestEnterRoomRejectsACharacterAlreadyInAnotherRoom pins that the seat cannot
// steal a character out of a room they already occupy.
func TestEnterRoomRejectsACharacterAlreadyInAnotherRoom(t *testing.T) {
	p, e := testPendingRoom(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 200, miniroom.Trade); err != nil {
		t.Fatalf("create the visitor's own room: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	assertErrorEvent(t, e, errOtherRequests)
	own, _ := p.RoomForCharacter(200)
	if own.OwnerId() != 200 {
		t.Errorf("the visitor was moved out of their own room: %+v", own)
	}
}

// --- TEARDOWN ----------------------------------------------------------------

// TestTeardownCharacterRemovesTheRoomAndCancelsBothSides pins design §3.3.
func TestTeardownCharacterRemovesTheRoomAndCancelsBothSides(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeDifferentMap); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	for _, id := range []character.Id{100, 200} {
		if _, ok := p.RoomForCharacter(id); ok {
			t.Errorf("character %d still holds a room after teardown", id)
		}
	}
	cancelled := statusEvents[trademsg.CancelledEventBody](t, e, trademsg.StatusTypeCancelled)
	if len(cancelled) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(cancelled))
	}
	if cancelled[0].Body.Reason != ReasonTradeDifferentMap {
		t.Errorf("reason: got %s, want %s", cancelled[0].Body.Reason, ReasonTradeDifferentMap)
	}
	if cancelled[0].OwnerId != 100 || cancelled[0].VisitorId != 200 {
		t.Errorf("cancelled parties: got owner %d visitor %d, want 100/200", cancelled[0].OwnerId, cancelled[0].VisitorId)
	}
}

// TestTeardownCharacterLosesToSettlement pins FR-6.5: once the room is
// SETTLING, a cancel trigger is recorded and ignored.
func TestTeardownCharacterLosesToSettlement(t *testing.T) {
	p, e := testPendingRoom(t)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	room, _ = p.RoomForCharacter(100)
	if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return cur.WithState(StateSettling), nil
	}); err != nil {
		t.Fatalf("move to settling: %v", err)
	}

	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a settling room was torn down by a cancel trigger")
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
}

// TestTeardownCharacterWithoutARoomIsANoOp pins that the teardown consumers can
// fire for every character in the world without emitting noise.
func TestTeardownCharacterWithoutARoomIsANoOp(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if got := len(e.messages(t, trademsg.EnvEventTopicStatus)); got != 0 {
		t.Errorf("status events: got %d, want 0", got)
	}
}

// --- error mapping -----------------------------------------------------------

// TestRegistryErrorCodeMapsEverySentinel pins that no registry rejection reaches
// the client as an unexplained generic failure. The expectation table is checked
// against AllRegistryErrors, so a sentinel added to the registry without a
// deliberate mapping fails here rather than shipping silently under the default
// arm.
func TestRegistryErrorCodeMapsEverySentinel(t *testing.T) {
	want := map[error]string{
		ErrRoomNotFound: errRoomClosed,
		ErrOwnerHasRoom: errOtherRequests,
		ErrHandleInUse:  errOtherRequests,
		ErrRoomFull:     errOtherRequests,
		ErrRoomFrozen:   errUnable,
	}

	for _, err := range AllRegistryErrors {
		expected, ok := want[err]
		if !ok {
			t.Errorf("registry sentinel %v has no expected mapping here; decide its enterError key in registryErrorCode and add it to this table", err)
			continue
		}
		if got := registryErrorCode(err); got != expected {
			t.Errorf("registryErrorCode(%v): got %s, want %s", err, got, expected)
		}
	}
	if len(want) != len(AllRegistryErrors) {
		t.Errorf("this table has %d entries but the registry declares %d sentinels", len(want), len(AllRegistryErrors))
	}
	if got := registryErrorCode(errors.New("something else")); got != errUnable {
		t.Errorf("registryErrorCode(unknown): got %s, want %s", got, errUnable)
	}
}

// TestEmitRollsBackTheOutboxOnFailure pins the atomicity the outbox buys: if the
// enclosing transaction fails, no partial batch is published.
func TestEmitRollsBackTheOutboxOnFailure(t *testing.T) {
	p, e := testProcessor(t)
	boom := errors.New("boom")
	err := p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		if putErr := mb.Put(trademsg.EnvEventTopicStatus, errorProvider(uuid.New(), testField(t), miniroom.Trade, 100, errUnable)); putErr != nil {
			return putErr
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("emit: got %v, want the closure's error", err)
	}
	if got := len(e.messages(t, trademsg.EnvEventTopicStatus)); got != 0 {
		t.Errorf("status events after a failed command: got %d, want 0", got)
	}
}

// compile-time assurance the fakes satisfy the seams they stand in for.
var (
	_ characterProvider = (*fakeCharacters)(nil)
	_ mapProvider       = (*fakeMaps)(nil)
	_ locationProvider  = (*fakeLocations)(nil)
	_ configProvider    = fakeConfig{}
)
