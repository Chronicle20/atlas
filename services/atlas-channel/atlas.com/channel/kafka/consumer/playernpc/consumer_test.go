package playernpc

import (
	"atlas-channel/playernpc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapc "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// playerNpcAttrJSON is one player-npcs JSON:API resource, mirroring
// kafka/consumer/map/player_npc_test.go's fixture of the same name.
func playerNpcAttrJSON(id string, objectId, scriptId uint32) string {
	return fmt.Sprintf(`{
		"type": "player-npcs",
		"id": "%s",
		"attributes": {
			"characterId": 1,
			"name": "Statue",
			"worldId": 0,
			"mapId": 100000000,
			"scriptId": %d,
			"objectId": %d,
			"gender": 0,
			"skin": 0,
			"face": 20000,
			"hair": 30000,
			"jobId": 100,
			"x": 100,
			"cy": 200,
			"fh": 1,
			"rx0": 50,
			"rx1": 150,
			"dir": 0,
			"worldRank": 1,
			"overallRank": 1,
			"worldJobRank": 1,
			"overallJobRank": 1,
			"equipment": [],
			"deployedAt": "2024-01-01T00:00:00Z"
		}
	}`, id, scriptId, objectId)
}

// stubPlayerNpcListServer stands up a fake atlas-player-npcs list endpoint
// returning the given fixed set of resources, for handleRepositioned's
// current-state read-back.
func stubPlayerNpcListServer(t *testing.T, entries ...string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(entries, ","))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(playernpc.SetBaseURLForTest(srv.URL))
}

func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func newTestServer(t *testing.T, ctx context.Context, worldId world.Id, channelId channel.Id) server.Model {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	ch := channel.NewModel(worldId, channelId)
	return server.NewProcessor(logrus.New(), ctx).Register(ten, ch, "127.0.0.1", 8484)
}

// addFieldSession registers a session in ctx's tenant registry with the given
// character id and field -- mirrors kafka/consumer/map/consumer_test.go's
// helper of the same name.
func addFieldSession(t *testing.T, ctx context.Context, l logrus.FieldLogger, characterId uint32, f field.Model) {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	sp := session.NewProcessor(l, ctx)
	sp.SetCharacterId(sessionId, characterId)
	sp.SetField(sessionId, f)
}

// announceCall is one recorded call to the announce seam: the writer name
// and the recipient's character id (broadcastSpawn/ForSessionsInMap fan out
// to every session in the map on its own goroutine, so cross-session order
// between announceCalls is not defined -- only the sequence of calls made
// to the SAME recipient, within one handler invocation, is).
type announceCall struct {
	writerName  string
	characterId uint32
}

// stubAnnounce swaps the package's announce seam for a recording stub,
// capturing each call's writer name and recipient character id. ForSessionsInMap
// fans out per-session announce calls across goroutines (production
// behavior, by design), so appends are mutex-guarded here; the resulting
// slice's element order only reflects real ordering within one recipient's
// own calls, never across recipients.
func stubAnnounce(t *testing.T) *[]announceCall {
	t.Helper()
	var mu sync.Mutex
	var seen []announceCall
	orig := announce
	announce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, s session.Model) error {
		mu.Lock()
		seen = append(seen, announceCall{writerName: writerName, characterId: s.CharacterId()})
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() { announce = orig })
	return &seen
}

// callsFor filters calls to the recipient with the given character id,
// preserving the order they were recorded -- meaningful because all calls
// to one recipient happen sequentially on that recipient's own goroutine.
func callsFor(calls []announceCall, characterId uint32) []string {
	var out []string
	for _, c := range calls {
		if c.characterId == characterId {
			out = append(out, c.writerName)
		}
	}
	return out
}

func statusModelFixture(objectId, scriptId uint32, worldId world.Id, mapId mapc.Id) StatusModel {
	return StatusModel{
		Id:          uuid.New(),
		CharacterId: 1,
		Name:        "Statue",
		WorldId:     byte(worldId),
		MapId:       uint32(mapId),
		ScriptId:    scriptId,
		ObjectId:    objectId,
		Gender:      0,
		Skin:        0,
		Face:        20000,
		Hair:        30000,
		JobId:       100,
		X:           100,
		Cy:          200,
		Fh:          1,
		Rx0:         50,
		Rx1:         150,
		Dir:         0,
		Equipment:   []StatusEquipment{},
		DeployedAt:  time.Now(),
	}
}

// TestPlayerNpcStatusConsumer covers plan.md Task 19's four event-type
// assertions for EVENT_TOPIC_PLAYER_NPC_STATUS.
func TestPlayerNpcStatusConsumer(t *testing.T) {
	l := logrus.New()
	const worldId = world.Id(0)
	const channelId = channel.Id(0)
	const mapId = mapc.Id(100000000)

	t.Run("DEPLOYED: SpawnNPC then ImitatedNPCData to everyone on the map", func(t *testing.T) {
		ctx := newTestCtx(t)
		ten := tenant.MustFromContext(ctx)
		defer session.ClearRegistryForTenant(ten.Id())
		f := field.NewBuilder(worldId, channelId, mapId).Build()
		addFieldSession(t, ctx, l, 1, f)
		addFieldSession(t, ctx, l, 2, f)
		sc := newTestServer(t, ctx, worldId, channelId)
		calls := stubAnnounce(t)

		e := StatusEvent[StatusModel]{Type: EventTypeDeployed, Body: statusModelFixture(100001, 9900001, worldId, mapId)}
		handleDeployed(sc, nil)(l, ctx, e)

		if len(*calls) != 4 {
			t.Fatalf("announce calls = %d, want 4 (spawn+imitated to 2 sessions)", len(*calls))
		}
		spawnCount, imitatedCount := 0, 0
		for _, c := range *calls {
			switch c.writerName {
			case npcpkt.NpcSpawnWriter:
				spawnCount++
			case npcpkt.NpcImitatedDataWriter:
				imitatedCount++
			}
		}
		if spawnCount != 2 || imitatedCount != 2 {
			t.Fatalf("spawn=%d imitated=%d, want 2/2 (one pair per session)", spawnCount, imitatedCount)
		}
		// Ordering within one recipient is guaranteed (broadcastSpawn issues
		// spawn-then-imitated on that recipient's own goroutine); ordering
		// across the two recipients (characterId 1 vs 2) is not, since
		// ForSessionsInMap fans them out concurrently -- so check each
		// recipient's own sequence rather than the combined slice's.
		for _, characterId := range []uint32{1, 2} {
			got := callsFor(*calls, characterId)
			want := []string{npcpkt.NpcSpawnWriter, npcpkt.NpcImitatedDataWriter}
			if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("calls to characterId %d = %v, want %v", characterId, got, want)
			}
		}
	})

	t.Run("DEPLOYED: world mismatch is ignored", func(t *testing.T) {
		ctx := newTestCtx(t)
		ten := tenant.MustFromContext(ctx)
		defer session.ClearRegistryForTenant(ten.Id())
		f := field.NewBuilder(worldId, channelId, mapId).Build()
		addFieldSession(t, ctx, l, 1, f)
		sc := newTestServer(t, ctx, worldId, channelId)
		calls := stubAnnounce(t)

		e := StatusEvent[StatusModel]{Type: EventTypeDeployed, Body: statusModelFixture(100001, 9900001, world.Id(99), mapId)}
		handleDeployed(sc, nil)(l, ctx, e)

		if len(*calls) != 0 {
			t.Fatalf("announce calls = %v, want none for a world mismatch", *calls)
		}
	})

	t.Run("UPDATED: ImitatedNPCData only, no despawn/respawn", func(t *testing.T) {
		ctx := newTestCtx(t)
		ten := tenant.MustFromContext(ctx)
		defer session.ClearRegistryForTenant(ten.Id())
		f := field.NewBuilder(worldId, channelId, mapId).Build()
		addFieldSession(t, ctx, l, 1, f)
		sc := newTestServer(t, ctx, worldId, channelId)
		calls := stubAnnounce(t)

		e := StatusEvent[StatusModel]{Type: EventTypeUpdated, Body: statusModelFixture(100001, 9900001, worldId, mapId)}
		handleUpdated(sc, nil)(l, ctx, e)

		if len(*calls) != 1 || (*calls)[0].writerName != npcpkt.NpcImitatedDataWriter {
			t.Fatalf("calls = %v, want exactly [%s]", *calls, npcpkt.NpcImitatedDataWriter)
		}
	})

	t.Run("REMOVED: RemoveNPC for the object id, to everyone on the map", func(t *testing.T) {
		ctx := newTestCtx(t)
		ten := tenant.MustFromContext(ctx)
		defer session.ClearRegistryForTenant(ten.Id())
		f := field.NewBuilder(worldId, channelId, mapId).Build()
		addFieldSession(t, ctx, l, 1, f)
		addFieldSession(t, ctx, l, 2, f)
		sc := newTestServer(t, ctx, worldId, channelId)
		calls := stubAnnounce(t)

		e := StatusEvent[StatusRemovedBody]{Type: EventTypeRemoved, Body: StatusRemovedBody{
			Id: uuid.New(), ObjectId: 100001, MapId: uint32(mapId), WorldId: byte(worldId),
		}}
		handleRemoved(sc, nil)(l, ctx, e)

		if len(*calls) != 2 {
			t.Fatalf("announce calls = %d, want 2 (one RemoveNPC per session)", len(*calls))
		}
		for _, c := range *calls {
			if c.writerName != npcpkt.NpcRemoveWriter {
				t.Fatalf("calls = %v, want only [%s]", *calls, npcpkt.NpcRemoveWriter)
			}
		}
	})

	t.Run("REPOSITIONED: per object RemoveNPC then SpawnNPC, then one ImitatedNPCData for the whole list", func(t *testing.T) {
		ctx := newTestCtx(t)
		ten := tenant.MustFromContext(ctx)
		defer session.ClearRegistryForTenant(ten.Id())
		f := field.NewBuilder(worldId, channelId, mapId).Build()
		addFieldSession(t, ctx, l, 1, f)
		sc := newTestServer(t, ctx, worldId, channelId)

		stubPlayerNpcListServer(t,
			playerNpcAttrJSON(uuid.New().String(), 100001, 9900001),
			playerNpcAttrJSON(uuid.New().String(), 100002, 9900002),
		)
		calls := stubAnnounce(t)

		e := StatusEvent[StatusRepositionedBody]{Type: EventTypeRepositioned, Body: StatusRepositionedBody{
			WorldId: byte(worldId), MapId: uint32(mapId),
			Npcs: []StatusRepositionedNpc{
				{Id: uuid.New(), ObjectId: 100001, X: 10, Cy: 20, Fh: 1, Rx0: 0, Rx1: 100},
				{Id: uuid.New(), ObjectId: 100002, X: 30, Cy: 40, Fh: 1, Rx0: 0, Rx1: 100},
			},
		}}
		handleRepositioned(sc, nil)(l, ctx, e)

		// One session in the map: 2 objects * (Remove + Spawn) + 1 batched
		// ImitatedNPCData = 5 announce calls.
		if len(*calls) != 5 {
			t.Fatalf("announce calls = %v (%d), want 5", *calls, len(*calls))
		}
		// Only one session is in the map here, so the entire sequence is on
		// that one recipient's own goroutine and its order is deterministic
		// (unlike the DEPLOYED/REMOVED cases with multiple recipients).
		want := []string{
			npcpkt.NpcRemoveWriter, npcpkt.NpcSpawnWriter,
			npcpkt.NpcRemoveWriter, npcpkt.NpcSpawnWriter,
			npcpkt.NpcImitatedDataWriter,
		}
		for i, w := range want {
			if (*calls)[i].writerName != w {
				t.Fatalf("calls[%d] = %s, want %s (full sequence %v)", i, (*calls)[i].writerName, w, *calls)
			}
		}
	})
}
