package _map

import (
	_map3 "atlas-channel/kafka/message/map"
	"atlas-channel/playernpc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	controllernpc "atlas-channel/npc/controller"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// playerNpcAttrJSON is one player-npcs JSON:API resource with a distinct
// objectId/scriptId, mirroring playernpc/rest_test.go's fixture shape.
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
// returning the given fixed set of resources and points playernpc's base URL
// at it for the test's duration.
func stubPlayerNpcListServer(t *testing.T, entries ...string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(entries, ","))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(playernpc.SetBaseURLForTest(srv.URL))
}

// stubPlayerNpcAnnounce swaps playerNpcAnnounce for a recording stub that
// captures the writer name (in call order) and the entry count of any
// ImitatedNpcData it sees, without touching a real socket -- mirrors
// consumer_test.go's stubDoorAnnounceForVisuals.
func stubPlayerNpcAnnounce(t *testing.T) (calls *[]string, imitatedCount *int) {
	t.Helper()
	var seen []string
	var lastImitatedCount int
	orig := playerNpcAnnounce
	playerNpcAnnounce = func(l logrus.FieldLogger, ctx context.Context, _ writer.Producer, writerName string, enc packet.Encode, _ session.Model) error {
		seen = append(seen, writerName)
		if writerName == npcpkt.NpcImitatedDataWriter {
			b := enc(l, ctx)(map[string]interface{}{})
			if len(b) > 0 {
				lastImitatedCount = int(b[0])
			}
		}
		return nil
	}
	t.Cleanup(func() { playerNpcAnnounce = orig })
	return &seen, &lastImitatedCount
}

// TestSpawnPlayerNpcForSession covers plan.md Task 19's four assertions for
// the map-enter Player NPC spawn path (FR-7.1/FR-7.4, design D-4).
func TestSpawnPlayerNpcForSession(t *testing.T) {
	l := logrus.New()

	t.Run("ordering: SpawnNPC strictly before ImitatedNPCData", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), 100001, 9900001))
		calls, imitatedCount := stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		if len(*calls) != 2 || (*calls)[0] != npcpkt.NpcSpawnWriter || (*calls)[1] != npcpkt.NpcImitatedDataWriter {
			t.Fatalf("writer calls = %v, want [%s %s]", *calls, npcpkt.NpcSpawnWriter, npcpkt.NpcImitatedDataWriter)
		}
		if *imitatedCount != 1 {
			t.Fatalf("ImitatedNpcData entry count = %d, want 1", *imitatedCount)
		}
	})

	t.Run("no controller grant is ever written for a Player NPC", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), 100002, 9900002))
		calls, _ := stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		for _, c := range *calls {
			if c == npcpkt.NpcSpawnRequestControllerWriter {
				t.Fatalf("SpawnNPCRequestController was written for a Player NPC; want none (FR-7.4)")
			}
		}
	})

	t.Run("batched: three Player NPCs produce three SpawnNPC and one ImitatedNPCData(3)", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubPlayerNpcListServer(t,
			playerNpcAttrJSON(uuid.New().String(), 100003, 9900003),
			playerNpcAttrJSON(uuid.New().String(), 100004, 9900004),
			playerNpcAttrJSON(uuid.New().String(), 100005, 9900005),
		)
		calls, imitatedCount := stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		spawnCount := 0
		imitatedWrites := 0
		for _, c := range *calls {
			switch c {
			case npcpkt.NpcSpawnWriter:
				spawnCount++
			case npcpkt.NpcImitatedDataWriter:
				imitatedWrites++
			}
		}
		if spawnCount != 3 {
			t.Errorf("SpawnNPC count = %d, want 3", spawnCount)
		}
		if imitatedWrites != 1 {
			t.Errorf("ImitatedNPCData write count = %d, want 1", imitatedWrites)
		}
		if *imitatedCount != 3 {
			t.Errorf("ImitatedNPCData entry count = %d, want 3", *imitatedCount)
		}
	})
}

// TestExitProducesNoControllerHandoffForPlayerNpc asserts design §7.3's
// claim directly against npc/controller's real (miniredis-backed) registry,
// not by assumption: a Player NPC's object id is NEVER passed to TryClaim,
// so it can never appear in ReleaseFor's result -- and the full
// CHARACTER_EXIT handler, run end to end, produces no panic/error while
// releasing an ordinary NPC's claim held by the same exiting character.
func TestExitProducesNoControllerHandoffForPlayerNpc(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()
	ten := tenant.MustFromContext(ctx)

	const exitingCharacterId uint32 = 42
	const ordinaryNpcObjectId uint32 = 500
	const playerNpcObjectId uint32 = 100006

	// TryClaim's hidden-winner-check fetches the candidate's buffs
	// (character/buff.GetByCharacterId); stub it to an empty list so the
	// claim resolves without retry-noise from an unconfigured service URL.
	buffsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(buffsSrv.Close)
	t.Setenv("BUFFS_SERVICE_URL", buffsSrv.URL+"/")

	// Deploy the Player NPC via the real spawn path -- it must never reach
	// the controller registry.
	stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), playerNpcObjectId, 9900006))
	_, _ = stubPlayerNpcAnnounce(t)
	if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
		t.Fatalf("spawnPlayerNpcsForSession: %v", err)
	}

	// Real (miniredis-backed) controller registry, isolated to this test.
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = rc.Close() }()
	controllernpc.InitRegistry(rc)

	cp := controllernpc.NewProcessor(l, ctx)
	won, cerr := cp.TryClaim(f, ordinaryNpcObjectId, exitingCharacterId)
	if cerr != nil || !won {
		t.Fatalf("TryClaim(ordinary npc) = won=%v err=%v, want won=true", won, cerr)
	}

	// The Player NPC's object id must never have entered the registry --
	// this is the structural guarantee spawnPlayerNpcsForSession relies on
	// (it never calls TryClaim), verified here against the real registry
	// rather than assumed.
	if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, playerNpcObjectId); oerr != nil || ok {
		t.Fatalf("ControllerOf(playerNpc) = ok=%v err=%v, want ok=false (never claimed)", ok, oerr)
	}

	// Full CHARACTER_EXIT handler, end to end. With only the exiting
	// character present in the field, ElectFor has no eligible candidate
	// (it excludes the exiting character), so no controller hand-off is
	// announced for ANY npc -- and running the handler on a field that also
	// has a deployed, never-claimed Player NPC id must not panic or error.
	addFieldSession(t, ctx, l, exitingCharacterId, f)
	ch := channel.NewModel(f.WorldId(), f.ChannelId())
	sc := server.NewProcessor(l, ctx).Register(ten, ch, "127.0.0.1", 8484)
	e := _map3.StatusEvent[_map3.CharacterExit]{
		TransactionId: uuid.New(),
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          _map3.EventTopicMapStatusTypeCharacterExit,
		Body:          _map3.CharacterExit{CharacterId: exitingCharacterId},
	}
	handleStatusEventCharacterExit(sc, nil)(l, ctx, e)

	// The ordinary NPC's claim must have been released by the handler
	// (proving it actually ran the release path), while the Player NPC
	// remains -- as always -- outside the registry entirely.
	if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, ordinaryNpcObjectId); oerr != nil || ok {
		t.Fatalf("ControllerOf(ordinaryNpc) after exit = ok=%v err=%v, want ok=false (released)", ok, oerr)
	}
	if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, playerNpcObjectId); oerr != nil || ok {
		t.Fatalf("ControllerOf(playerNpc) after exit = ok=%v err=%v, want ok=false (never entered)", ok, oerr)
	}
}
