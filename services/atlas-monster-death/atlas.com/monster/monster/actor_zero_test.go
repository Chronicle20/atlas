package monster

// actor_zero_test.go pins the ActorId = 0 (no-killer) detonation behaviour
// that design.md §5/§8.2 and PRD FR-6.4 assume: drops spawn unowned without
// error, quest-specific drops are excluded (since a killerId of 0 never has
// any started quests), and no EXP is distributed. It is a characterization
// task -- these tests must pass against the existing production code
// unmodified.

import (
	"atlas-monster-death/monster/drop"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// TestFilterByQuestStateExcludesQuestDropsForNoKiller pins that a killerId
// of 0 (no killer, e.g. a self-destruct detonation) excludes every
// quest-specific drop, regardless of what the quest service says (or
// whether it is reachable at all).
func TestFilterByQuestStateExcludesQuestDropsForNoKiller(t *testing.T) {
	nonQuestDrop, err := drop.NewBuilder().SetItemId(1000).SetChance(999999).SetMinimumQuantity(1).SetMaximumQuantity(1).SetQuestId(0).Build()
	if err != nil {
		t.Fatal(err)
	}
	questDrop, err := drop.NewBuilder().SetItemId(2000).SetChance(999999).SetMinimumQuantity(1).SetMaximumQuantity(1).SetQuestId(4000).Build()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		drops   []drop.Model
		handler http.HandlerFunc
	}{
		{
			name:  "quest lookup errors",
			drops: []drop.Model{nonQuestDrop, questDrop},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		},
		{
			name:  "character has no started quests",
			drops: []drop.Model{nonQuestDrop, questDrop},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				_, _ = w.Write([]byte(`{"data":[],"meta":{"total":0,"page":{"number":1,"size":250,"last":1}}}`))
			},
		},
		{
			name:  "no quest drops at all",
			drops: []drop.Model{nonQuestDrop},
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("quest server must not be hit when no drop has a non-zero questId")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

			ctx := newTestContext(t)
			l, _ := test.NewNullLogger()

			p := &ProcessorImpl{l: l, ctx: ctx}
			got := p.filterByQuestState(0, tc.drops)

			gotIds := make([]uint32, 0, len(got))
			for _, d := range got {
				gotIds = append(gotIds, d.ItemId())
			}
			if len(gotIds) != 1 || gotIds[0] != 1000 {
				t.Fatalf("expected only item 1000 to survive filtering, got %v", gotIds)
			}
		})
	}
}

// TestCreateDropsWithNoKillerSpawnsUnownedDrop pins that a detonation with
// killerId 0 spawns its drop unowned (ownerId 0, ownerPartyId 0) rather than
// erroring -- rates fall back to Default() and party lookup fails silently,
// exactly as they would for any character not in a party or on default rates.
func TestCreateDropsWithNoKillerSpawnsUnownedDrop(t *testing.T) {
	dropSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"drops","attributes":{"itemId":1000,"minimumQuantity":1,"maximumQuantity":1,"questId":0,"chance":999999}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
	}))
	defer dropSrv.Close()
	t.Setenv("DROPS_INFORMATION_SERVICE_URL", dropSrv.URL+"/")

	ratesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ratesSrv.Close()
	t.Setenv("RATES_SERVICE_URL", ratesSrv.URL+"/")

	partySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer partySrv.Close()
	t.Setenv("PARTIES_SERVICE_URL", partySrv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(211000000)).SetInstance(uuid.Nil).Build()

	emitted.Reset()
	err := NewProcessor(l, ctx).CreateDrops(f, 500, 8500003, 10, 20, 0)
	if err != nil {
		t.Fatalf("expected nil error for a no-killer detonation, got %v", err)
	}

	msgs := emitted.Messages(drop.EnvCommandTopic)
	if len(msgs) != 1 {
		t.Fatalf("expected exactly one spawn drop command, got %d", len(msgs))
	}

	var out struct {
		Body struct {
			OwnerId      uint32 `json:"ownerId"`
			OwnerPartyId uint32 `json:"ownerPartyId"`
			ItemId       uint32 `json:"itemId"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &out); err != nil {
		t.Fatalf("unable to unmarshal spawn drop command: %v", err)
	}
	if out.Body.OwnerId != 0 {
		t.Fatalf("expected unowned drop (ownerId 0), got %d", out.Body.OwnerId)
	}
	if out.Body.OwnerPartyId != 0 {
		t.Fatalf("expected no owning party (ownerPartyId 0), got %d", out.Body.OwnerPartyId)
	}
	if out.Body.ItemId != 1000 {
		t.Fatalf("expected the configured drop itemId 1000, got %d", out.Body.ItemId)
	}
}

// TestCalculateExperienceStandardDeviationThresholdEmptyIsNaN pins a known
// adjacent defect (design §8.2): calculateExperienceStandardDeviationThreshold
// divides by totalEntries (and by len(entryExperienceRatio)) unconditionally,
// so an empty distribution produces NaN via 0/0. It is harmless today because
// the only caller (isWhiteExperienceGain via produceDistribution) then
// iterates an empty personalRatio map, and harmless on the ActorId = 0 path
// for the same reason -- pinned here, not changed.
func TestCalculateExperienceStandardDeviationThresholdEmptyIsNaN(t *testing.T) {
	got := calculateExperienceStandardDeviationThreshold([]float64{}, 0)
	if !math.IsNaN(got) {
		t.Fatalf("expected NaN for an empty distribution, got %f", got)
	}
}

// TestDistributeExperienceWithEmptyEntriesIsNoOp pins that a no-killer
// detonation (damageEntries nil) distributes no EXP: DistributeExperience
// must return nil and must never call the character service.
func TestDistributeExperienceWithEmptyEntriesIsNoOp(t *testing.T) {
	dataSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"8500003","type":"monsters","attributes":{"hp":4000,"experience":150}}}`))
	}))
	defer dataSrv.Close()
	t.Setenv("DATA_SERVICE_URL", dataSrv.URL+"/")

	mapsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"total":0,"page":{"number":1,"size":250,"last":1}}}`))
	}))
	defer mapsSrv.Close()
	t.Setenv("MAPS_SERVICE_URL", mapsSrv.URL+"/")

	charSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("character service must not be called when there are no damage entries")
	}))
	defer charSrv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", charSrv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(211000000)).SetInstance(uuid.Nil).Build()

	err := NewProcessor(l, ctx).DistributeExperience(f, 8500003, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
