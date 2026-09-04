package quest_test

import (
	"atlas-saga-orchestrator/quest"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	questmsg "atlas-saga-orchestrator/kafka/message/quest"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var emitted *producertest.Capture

// TestMain installs a capturing producer so RequestExplorerQuest's
// force-start step (a real Kafka emit via producer.ProviderImpl) succeeds
// instantly instead of retrying against an unreachable broker for ~42s (see
// producertest package doc).
func TestMain(m *testing.M) {
	if err := os.Setenv(string(questmsg.EnvCommandTopic), string(questmsg.EnvCommandTopic)); err != nil {
		panic(err)
	}
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// medalMapResponseDoc renders a JSON:API "medal-maps" single-resource
// response mirroring atlas-quest's POST .../medal-maps reply.
func medalMapResponseDoc(count uint32, newlyRecorded bool) string {
	b, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"id":   "1",
			"type": "medal-maps",
			"attributes": map[string]any{
				"count":         count,
				"newlyRecorded": newlyRecorded,
			},
		},
	})
	return string(b)
}

// questDataResponseDoc renders a JSON:API "quests" single-resource response
// mirroring atlas-data's GET /data/quests/{questId} reply, carrying only the
// endRequirements fields explorerQuest reads.
func questDataResponseDoc(questId uint32, infoNumber uint32, infoEx []string) string {
	b, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"id":   itoa(questId),
			"type": "quests",
			"attributes": map[string]any{
				"endRequirements": map[string]any{
					"infoNumber": infoNumber,
					"infoEx":     infoEx,
				},
			},
		},
	})
	return string(b)
}

func itoa(v uint32) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// TestRequestExplorerQuest_NewlyRecordedResolvesThreshold proves that when
// atlas-quest reports a newly-recorded medal map, RequestExplorerQuest loads
// the quest's infoNumber/infoEx threshold from atlas-data (falling back to
// questId per Cosmic's Quest.java:268-270 when infoNumber is 0) and force-
// starts the quest via npc 9000066 on COMMAND_TOPIC_QUEST -- Cosmic's
// quest.forceStart(getPlayer(), 9000066) (MapScriptMethods.java:104-139).
func TestRequestExplorerQuest_NewlyRecordedResolvesThreshold(t *testing.T) {
	emitted.Reset()

	const characterId = uint32(100100)
	const questId = uint32(29000)
	const mapId = uint32(101000000)

	medalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(medalMapResponseDoc(3, true)))
	}))
	defer medalSrv.Close()
	t.Setenv("QUESTS_SERVICE_URL", medalSrv.URL+"/")

	var questDataCalled bool
	dataSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		questDataCalled = true
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(questDataResponseDoc(questId, 0, []string{"5"})))
	}))
	defer dataSrv.Close()
	t.Setenv("DATA_SERVICE_URL", dataSrv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	p := quest.NewProcessor(l, ctx)
	got, err := p.RequestExplorerQuest(uuid.New(), world.Id(0), characterId, questId, mapId)
	if err != nil {
		t.Fatalf("RequestExplorerQuest returned error: %v", err)
	}

	if !questDataCalled {
		t.Fatal("expected quest-data lookup on a newly-recorded medal map")
	}
	if !got.NewlyRecorded {
		t.Fatal("expected NewlyRecorded=true")
	}
	if got.Count != 3 {
		t.Fatalf("expected Count=3, got %d", got.Count)
	}
	if got.InfoNumber != questId {
		t.Fatalf("expected InfoNumber to fall back to questId [%d], got %d", questId, got.InfoNumber)
	}
	if got.Threshold != 5 {
		t.Fatalf("expected Threshold=5, got %d", got.Threshold)
	}

	msgs := emitted.Messages(string(questmsg.EnvCommandTopic))
	if len(msgs) != 1 {
		t.Fatalf("expected 1 start-quest command, got %d", len(msgs))
	}
	var cmd questmsg.Command[questmsg.StartCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unable to unmarshal start command: %v", err)
	}
	if cmd.Type != questmsg.CommandTypeStart {
		t.Fatalf("expected command type %s, got %s", questmsg.CommandTypeStart, cmd.Type)
	}
	if cmd.CharacterId != characterId {
		t.Fatalf("expected characterId %d, got %d", characterId, cmd.CharacterId)
	}
	if cmd.Body.QuestId != questId {
		t.Fatalf("expected questId %d, got %d", questId, cmd.Body.QuestId)
	}
	if cmd.Body.NpcId != 9000066 {
		t.Fatalf("expected force-start npcId 9000066, got %d", cmd.Body.NpcId)
	}
}

// TestRequestExplorerQuest_AlreadyRecordedSkipsProgressWrite proves the dedup
// path Cosmic implements with `if (!qs.addMedalMap(...)) return;`
// (MapScriptMethods.java:104-139): when atlas-quest reports the map was
// already recorded, RequestExplorerQuest returns without ever calling
// atlas-data to resolve a threshold.
func TestRequestExplorerQuest_AlreadyRecordedSkipsProgressWrite(t *testing.T) {
	emitted.Reset()

	const characterId = uint32(100100)
	const questId = uint32(29000)
	const mapId = uint32(101000000)

	medalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(medalMapResponseDoc(7, false)))
	}))
	defer medalSrv.Close()
	t.Setenv("QUESTS_SERVICE_URL", medalSrv.URL+"/")

	var questDataCalled bool
	dataSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		questDataCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer dataSrv.Close()
	t.Setenv("DATA_SERVICE_URL", dataSrv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	p := quest.NewProcessor(l, ctx)
	got, err := p.RequestExplorerQuest(uuid.New(), world.Id(0), characterId, questId, mapId)
	if err != nil {
		t.Fatalf("RequestExplorerQuest returned error: %v", err)
	}

	if questDataCalled {
		t.Fatal("expected no quest-data lookup when the medal map was already recorded")
	}
	if got.NewlyRecorded {
		t.Fatal("expected NewlyRecorded=false")
	}
	if got.Count != 7 {
		t.Fatalf("expected Count=7, got %d", got.Count)
	}

	msgs := emitted.Messages(string(questmsg.EnvCommandTopic))
	if len(msgs) != 1 {
		t.Fatalf("expected 1 start-quest command (force-start still happens on the dedup path), got %d", len(msgs))
	}
}
