package _map

import (
	npcKafka "atlas-channel/kafka/message/npc"
	scriptednpc "atlas-channel/map/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
)

// npcAnnounce is one scriptedNpcAnnounce invocation captured by
// stubScriptedNpcAnnounce.
type npcAnnounce struct {
	Writer string
	Body   []byte
}

// npcAnnounceRecorder collects npcAnnounce invocations made concurrently --
// handleStatusEventNpcCreated fans the announce out through
// map.(*ProcessorImpl).ForSessionsInMap, one goroutine per session -- mirrors
// jukeboxAnnounceRecorder.
type npcAnnounceRecorder struct {
	mu   sync.Mutex
	seen []npcAnnounce
}

func (r *npcAnnounceRecorder) record(a npcAnnounce) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, a)
}

func (r *npcAnnounceRecorder) snapshot() []npcAnnounce {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]npcAnnounce, len(r.seen))
	copy(out, r.seen)
	return out
}

// stubScriptedNpcAnnounce swaps scriptedNpcAnnounce for a recording stub,
// without touching a real socket writer -- mirrors stubDoorAnnounceForJukebox.
func stubScriptedNpcAnnounce(t *testing.T) (restore func(), rec *npcAnnounceRecorder) {
	t.Helper()
	rec = &npcAnnounceRecorder{}

	orig := scriptedNpcAnnounce
	scriptedNpcAnnounce = func(l logrus.FieldLogger, ctx context.Context, _ writer.Producer, writerName string, enc packet.Encode, _ session.Model) error {
		rec.record(npcAnnounce{Writer: writerName, Body: enc(l, ctx)(nil)})
		return nil
	}

	return func() { scriptedNpcAnnounce = orig }, rec
}

// TestScriptedNpcSpawn_EncodesTheModelsFields asserts the wire body carries
// the uniqueId/npcId/x/y fields map/npc.Model actually holds -- the packet
// layer (libs/atlas-packet/npc/clientbound/spawn.go) is exercised through
// the real Encode, not re-implemented here.
func TestScriptedNpcSpawn_EncodesTheModelsFields(t *testing.T) {
	l := logrus.New()
	spawn := ScriptedNpcSpawn(42, 1104100, 2830, 78, 5)
	got := spawn.Encode(l, context.Background())(nil)

	if len(got) < 12 {
		t.Fatalf("encoded body too short: %d bytes", len(got))
	}
	gotUniqueId := binary.LittleEndian.Uint32(got[0:4])
	gotNpcId := binary.LittleEndian.Uint32(got[4:8])
	gotX := int16(binary.LittleEndian.Uint16(got[8:10]))
	gotY := int16(binary.LittleEndian.Uint16(got[10:12]))

	if gotUniqueId != 42 {
		t.Errorf("uniqueId = %d, want 42", gotUniqueId)
	}
	if gotNpcId != 1104100 {
		t.Errorf("npcId = %d, want 1104100", gotNpcId)
	}
	if gotX != 2830 {
		t.Errorf("x = %d, want 2830", gotX)
	}
	if gotY != 78 {
		t.Errorf("y (as cy) = %d, want 78", gotY)
	}
}

// TestHandleStatusEventNpcCreated_BroadcastsToEverySessionInField covers the
// bound path (task-BC brief): the CREATED status event reaches every session
// already in the field as SPAWN_NPC.
func TestHandleStatusEventNpcCreated_BroadcastsToEverySessionInField(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]{
		WorldId:   world.Id(0),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		UniqueId:  7,
		Type:      npcKafka.EventStatusTypeCreated,
		Body:      npcKafka.CreatedStatusEventBody{NpcId: 1104100, X: 2830, Y: 78, Fh: 5},
	}
	handleStatusEventNpcCreated(sc, nil)(l, ctx, e)

	calls := rec.snapshot()
	if len(calls) != 2 {
		t.Fatalf("announce count = %d, want 2", len(calls))
	}
	for _, c := range calls {
		if c.Writer != npcpkt.NpcSpawnWriter {
			t.Fatalf("writer = %s, want %s", c.Writer, npcpkt.NpcSpawnWriter)
		}
		gotUniqueId := binary.LittleEndian.Uint32(c.Body[0:4])
		if gotUniqueId != 7 {
			t.Fatalf("uniqueId = %d, want 7", gotUniqueId)
		}
	}
}

// TestHandleStatusEventNpcCreated_IgnoresOtherEventTypes pins the Type guard.
func TestHandleStatusEventNpcCreated_IgnoresOtherEventTypes(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]{
		WorldId:   world.Id(0),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		UniqueId:  7,
		Type:      "SOMETHING_ELSE",
		Body:      npcKafka.CreatedStatusEventBody{NpcId: 1104100, X: 2830, Y: 78, Fh: 5},
	}
	handleStatusEventNpcCreated(sc, nil)(l, ctx, e)

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

// TestHandleStatusEventNpcCreated_IgnoresOtherWorldChannel pins the sc.Is guard.
func TestHandleStatusEventNpcCreated_IgnoresOtherWorldChannel(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]{
		WorldId:   world.Id(1),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		UniqueId:  7,
		Type:      npcKafka.EventStatusTypeCreated,
		Body:      npcKafka.CreatedStatusEventBody{NpcId: 1104100, X: 2830, Y: 78, Fh: 5},
	}
	handleStatusEventNpcCreated(sc, nil)(l, ctx, e)

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

// TestSpawnScriptedNpcsForSession_SpawnsExistingScriptedNpc is the
// regression this task exists to prevent: a character entering a field
// that already has a scripted NPC (placed by an earlier spawn_npc, or by an
// onUserEnter action that already ran for an earlier character) receives a
// SPAWN_NPC packet for it.
func TestSpawnScriptedNpcsForSession_SpawnsExistingScriptedNpc(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "npcs",
					"id": "7",
					"attributes": {
						"npcId": 1104100,
						"x": 2830,
						"y": 78,
						"fh": 5
					}
				}
			]
		}`))
	}))
	defer srv.Close()
	defer scriptednpc.SetBaseURLForTest(srv.URL)()

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	s := session.NewSession(uuid.New(), ten, 0, nil)

	if err := spawnScriptedNpcsForSession(l, ctx, nil, s, f); err != nil {
		t.Fatalf("spawnScriptedNpcsForSession: %v", err)
	}

	calls := rec.snapshot()
	if len(calls) != 1 {
		t.Fatalf("announce count = %d, want 1", len(calls))
	}
	if calls[0].Writer != npcpkt.NpcSpawnWriter {
		t.Fatalf("writer = %s, want %s", calls[0].Writer, npcpkt.NpcSpawnWriter)
	}
	gotUniqueId := binary.LittleEndian.Uint32(calls[0].Body[0:4])
	gotNpcId := binary.LittleEndian.Uint32(calls[0].Body[4:8])
	if gotUniqueId != 7 {
		t.Errorf("uniqueId = %d, want 7", gotUniqueId)
	}
	if gotNpcId != 1104100 {
		t.Errorf("npcId = %d, want 1104100", gotNpcId)
	}
}

// TestSpawnScriptedNpcsForSession_EmptyFieldIsSilent asserts a field with
// no scripted NPCs (the overwhelmingly common case) announces nothing and
// returns nil -- not an error.
func TestSpawnScriptedNpcsForSession_EmptyFieldIsSilent(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()
	defer scriptednpc.SetBaseURLForTest(srv.URL)()

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	s := session.NewSession(uuid.New(), ten, 0, nil)

	if err := spawnScriptedNpcsForSession(l, ctx, nil, s, f); err != nil {
		t.Fatalf("spawnScriptedNpcsForSession: %v", err)
	}
	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

// TestSpawnScriptedNpcsForSession_BytesMatchBroadcastPath asserts the
// resync path (spawnScriptedNpcsForSession, via this test) and the broadcast
// path (handleStatusEventNpcCreated) put identical bytes on the wire for the
// same scripted NPC, since both build the packet through the one exported
// ScriptedNpcSpawn -- neither re-derives the rx0/rx1/facing/cy substitution.
func TestSpawnScriptedNpcsForSession_BytesMatchBroadcastPath(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "npcs",
					"id": "7",
					"attributes": {
						"npcId": 1104100,
						"x": 2830,
						"y": 78,
						"fh": 5
					}
				}
			]
		}`))
	}))
	defer srv.Close()
	defer scriptednpc.SetBaseURLForTest(srv.URL)()

	restore, rec := stubScriptedNpcAnnounce(t)
	defer restore()

	s := session.NewSession(uuid.New(), ten, 0, nil)
	if err := spawnScriptedNpcsForSession(l, ctx, nil, s, f); err != nil {
		t.Fatalf("spawnScriptedNpcsForSession: %v", err)
	}
	resyncCalls := rec.snapshot()
	if len(resyncCalls) != 1 {
		t.Fatalf("resync announce count = %d, want 1", len(resyncCalls))
	}

	broadcastSpawn := ScriptedNpcSpawn(7, 1104100, 2830, 78, 5)
	wantBody := broadcastSpawn.Encode(l, ctx)(nil)

	if string(resyncCalls[0].Body) != string(wantBody) {
		t.Fatalf("resync bytes = %v, want %v (broadcast path)", resyncCalls[0].Body, wantBody)
	}
}
