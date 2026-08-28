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
	"sync"
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

// stubBuffsServer points character/buff at an empty-buff-list stub, so
// TryClaim's GM-hidden winner-check resolves without retry noise from an
// unconfigured service URL.
func stubBuffsServer(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")
}

// indexOf returns the first position of v in s, or -1.
func indexOf(s []string, v string) int {
	for i, e := range s {
		if e == v {
			return i
		}
	}
	return -1
}

// testRegistryOnce guards the package-wide miniredis-backed controller
// registry. npc/controller.InitRegistry is a sync.Once singleton, so the
// first initializer in the test binary wins for the whole run -- tests share
// one registry and keep their object ids distinct rather than each standing
// up their own.
var testRegistryOnce sync.Once

// stubControllerRegistry ensures the real (miniredis-backed) controller
// registry is initialized for tests that assert claim/grant behaviour.
func stubControllerRegistry(t *testing.T) {
	t.Helper()
	testRegistryOnce.Do(func() {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("miniredis: %v", err)
		}
		controllernpc.InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	})
	if controllernpc.GetRegistry() == nil {
		t.Fatal("controller registry not initialized")
	}
}

// TestSpawnPlayerNpcForSession covers the map-enter Player NPC spawn path
// (FR-7.1) including the controller grant added by task-251 bug report §5.
func TestSpawnPlayerNpcForSession(t *testing.T) {
	l := logrus.New()

	t.Run("ordering: SpawnNPC strictly before ImitatedNPCData", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubBuffsServer(t)
		stubControllerRegistry(t)
		stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), 101001, 9901001))
		calls, imitatedCount := stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		spawnAt, imitatedAt := indexOf(*calls, npcpkt.NpcSpawnWriter), indexOf(*calls, npcpkt.NpcImitatedDataWriter)
		if spawnAt < 0 || imitatedAt < 0 || spawnAt > imitatedAt {
			t.Fatalf("writer calls = %v, want %s before %s", *calls, npcpkt.NpcSpawnWriter, npcpkt.NpcImitatedDataWriter)
		}
		if *imitatedCount != 1 {
			t.Fatalf("ImitatedNpcData entry count = %d, want 1", *imitatedCount)
		}
	})

	// task-251 bug report §5: the controller grant is what reaches
	// CNpc::SetActive client-side, and SetActive is what lets the NPC speak.
	// A Player NPC that wins its claim MUST receive the grant, and it must
	// arrive after its SpawnNPC (the client needs the object first).
	t.Run("controller grant follows SpawnNPC when the session wins the claim", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubBuffsServer(t)
		stubControllerRegistry(t)
		stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), 101002, 9901002))
		calls, _ := stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		want := []string{npcpkt.NpcSpawnWriter, npcpkt.NpcSpawnRequestControllerWriter, npcpkt.NpcImitatedDataWriter}
		if len(*calls) != len(want) {
			t.Fatalf("writer calls = %v, want %v", *calls, want)
		}
		for i, w := range want {
			if (*calls)[i] != w {
				t.Fatalf("writer calls = %v, want %v", *calls, want)
			}
		}
	})

	t.Run("Player NPC object id enters the controller registry", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		ten := tenant.MustFromContext(ctx)
		stubBuffsServer(t)
		stubControllerRegistry(t)
		stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), 101007, 9901007))
		_, _ = stubPlayerNpcAnnounce(t)

		if err := spawnPlayerNpcsForSession(l, ctx, nil, session.Model{}, f); err != nil {
			t.Fatalf("spawnPlayerNpcsForSession: %v", err)
		}

		if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, 101007); oerr != nil || !ok {
			t.Fatalf("ControllerOf(playerNpc) = ok=%v err=%v, want ok=true (claimed)", ok, oerr)
		}
	})

	t.Run("batched: three Player NPCs produce three SpawnNPC and one ImitatedNPCData(3)", func(t *testing.T) {
		ctx := newTestCtx(t)
		f := newTestField()
		stubBuffsServer(t)
		stubControllerRegistry(t)
		stubPlayerNpcListServer(t,
			playerNpcAttrJSON(uuid.New().String(), 101003, 9901003),
			playerNpcAttrJSON(uuid.New().String(), 101004, 9901004),
			playerNpcAttrJSON(uuid.New().String(), 101005, 9901005),
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

// TestExitReleasesPlayerNpcControllerEntry asserts the post-bug-report-§5
// contract against npc/controller's real (miniredis-backed) registry: a
// Player NPC's object id DOES enter the controller registry -- the grant is
// what lets the client run CNpc::DoActionOrChat -- and the ordinary
// CHARACTER_EXIT release path therefore reclaims it exactly as it does an
// ordinary NPC's, with no Player-NPC special-casing anywhere in the handler.
func TestExitReleasesPlayerNpcControllerEntry(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()
	ten := tenant.MustFromContext(ctx)

	const exitingCharacterId uint32 = 42
	const ordinaryNpcObjectId uint32 = 500
	const playerNpcObjectId uint32 = 101006

	stubBuffsServer(t)
	stubControllerRegistry(t)

	// Deploy the Player NPC via the real spawn path, as the exiting
	// character -- it claims control on the way in.
	stubPlayerNpcListServer(t, playerNpcAttrJSON(uuid.New().String(), playerNpcObjectId, 9901006))
	_, _ = stubPlayerNpcAnnounce(t)
	addFieldSession(t, ctx, l, exitingCharacterId, f)
	if err := session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(exitingCharacterId, func(s session.Model) error {
		return spawnPlayerNpcsForSession(l, ctx, nil, s, f)
	}); err != nil {
		t.Fatalf("spawnPlayerNpcsForSession: %v", err)
	}

	cp := controllernpc.NewProcessor(l, ctx)
	won, cerr := cp.TryClaim(f, ordinaryNpcObjectId, exitingCharacterId)
	if cerr != nil || !won {
		t.Fatalf("TryClaim(ordinary npc) = won=%v err=%v, want won=true", won, cerr)
	}

	// The Player NPC's object id IS in the registry, held by the character
	// that entered -- the inverse of the pre-fix FR-7.4 contract.
	cur, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, playerNpcObjectId)
	if oerr != nil || !ok {
		t.Fatalf("ControllerOf(playerNpc) = ok=%v err=%v, want ok=true (claimed on spawn)", ok, oerr)
	}
	if cur != exitingCharacterId {
		t.Fatalf("ControllerOf(playerNpc) = %d, want %d", cur, exitingCharacterId)
	}

	// Full CHARACTER_EXIT handler, end to end. With only the exiting
	// character present in the field, ElectFor has no eligible candidate
	// (it excludes the exiting character), so both NPCs are released and
	// left uncontrolled until the next enter. The session was already
	// registered above, for the spawn.
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

	// Both entries released by the same object-id-agnostic release path.
	if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, ordinaryNpcObjectId); oerr != nil || ok {
		t.Fatalf("ControllerOf(ordinaryNpc) after exit = ok=%v err=%v, want ok=false (released)", ok, oerr)
	}
	if _, ok, oerr := controllernpc.GetRegistry().ControllerOf(ctx, ten, f, playerNpcObjectId); oerr != nil || ok {
		t.Fatalf("ControllerOf(playerNpc) after exit = ok=%v err=%v, want ok=false (released)", ok, oerr)
	}
}
