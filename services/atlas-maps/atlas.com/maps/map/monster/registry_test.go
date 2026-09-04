package monster

import (
	"atlas-maps/map/character"
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	monster2 "atlas-maps/data/map/monster"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupSpawnTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), mr
}

// newTestRegistry builds a fully-initialized SpawnPointRegistry suitable for
// use in tests. Routing through newRegistry ensures the test registry's key
// shapes never drift from the singleton's.
func newTestRegistry(client *goredis.Client) *SpawnPointRegistry {
	return newRegistry(client)
}

func TestSpawnPointRegistry_FlushTenant_DeletesAllForTenant(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	tid := uuid.New()
	ctx := context.Background()
	l := logrus.New()

	te, err := tenant.Create(tid, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	for i := 0; i < 3; i++ {
		f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(uint32(100+i))).Build()
		mapKey := character.MapKey{Tenant: te, Field: f}
		if err := r.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
			{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: uint32(100 + i)}, NextSpawnAt: time.Now()},
		}); err != nil {
			t.Fatalf("SetSpawnPointsForMap seed: %v", err)
		}
	}

	deleted, err := r.FlushTenant(ctx, l, tid)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}
}

func TestSpawnPointRegistry_FlushTenant_TenantIsolation(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	tA := uuid.New()
	tB := uuid.New()
	ctx := context.Background()
	l := logrus.New()

	teA, err := tenant.Create(tA, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create A: %v", err)
	}
	teB, err := tenant.Create(tB, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create B: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(1)).Build()
	mapKeyA := character.MapKey{Tenant: teA, Field: f}
	mapKeyB := character.MapKey{Tenant: teB, Field: f}

	if err := r.SetSpawnPointsForMap(ctx, mapKeyA, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100}, NextSpawnAt: time.Now()},
	}); err != nil {
		t.Fatalf("SetSpawnPointsForMap A: %v", err)
	}
	if err := r.SetSpawnPointsForMap(ctx, mapKeyB, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100}, NextSpawnAt: time.Now()},
	}); err != nil {
		t.Fatalf("SetSpawnPointsForMap B: %v", err)
	}

	deleted, err := r.FlushTenant(ctx, l, tA)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	pts, ok := r.GetSpawnPointsForMap(ctx, mapKeyB)
	if !ok || len(pts) == 0 {
		t.Fatalf("tenant B's spawn key should still exist")
	}
}

func TestSpawnPointRegistry_FlushTenant_EmptyTenant(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	deleted, err := r.FlushTenant(context.Background(), logrus.New(), uuid.New())
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

// TestFlushTenant_MatchesWriteKeyUnderEnvPrefix reproduces the L296 bug:
// a write under <env>:atlas:maps:spawn:<bare-uuid>:... must be found and
// deleted by FlushTenant(tenantId) regardless of ATLAS_ENV.
func TestFlushTenant_MatchesWriteKeyUnderEnvPrefix(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)

	tid := uuid.New()
	te, err := tenant.Create(tid, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100100)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	r := newTestRegistry(client)
	if err := r.SetSpawnPointsForMap(context.Background(), mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: time.Now()},
	}); err != nil {
		t.Fatalf("SetSpawnPointsForMap: %v", err)
	}

	deleted, err := r.FlushTenant(context.Background(), logrus.New(), tid)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("FlushTenant deleted = %d, want 1 (scan/write key mismatch)", deleted)
	}
}

// mockSpawnDataProcessor implements monster2.Processor for InitializeForMap
// tests. GetSpawnPoints returns the unfiltered slice; Classify partitions it.
type mockSpawnDataProcessor struct {
	spawnPoints []monster2.SpawnPoint
}

func (m *mockSpawnDataProcessor) SpawnPointProvider(_ _map.Id) model.Provider[[]monster2.SpawnPoint] {
	return func() ([]monster2.SpawnPoint, error) { return m.spawnPoints, nil }
}

func (m *mockSpawnDataProcessor) GetSpawnPoints(_ _map.Id) ([]monster2.SpawnPoint, error) {
	return m.spawnPoints, nil
}

func TestInitializeForMap_PartitionsByMobTimeAndHide(t *testing.T) {
	cases := []struct {
		name          string
		points        []monster2.SpawnPoint
		wantRecurring int
		wantOneTime   int
	}{
		{
			name: "all one-time",
			points: func() []monster2.SpawnPoint {
				var pts []monster2.SpawnPoint
				for i := uint32(1); i <= 10; i++ {
					pts = append(pts, monster2.SpawnPoint{Id: i, MobTime: -1, Hide: false})
				}
				return pts
			}(),
			wantRecurring: 0,
			wantOneTime:   10,
		},
		{
			name: "recurring only",
			points: func() []monster2.SpawnPoint {
				var pts []monster2.SpawnPoint
				for i := uint32(1); i <= 4; i++ {
					pts = append(pts, monster2.SpawnPoint{Id: i, MobTime: 0, Hide: false})
				}
				return pts
			}(),
			wantRecurring: 4,
			wantOneTime:   0,
		},
		{
			name: "mixed",
			points: []monster2.SpawnPoint{
				{Id: 1, MobTime: 0, Hide: false},
				{Id: 2, MobTime: -1, Hide: false},
				{Id: 3, MobTime: 30, Hide: false},
			},
			wantRecurring: 2,
			wantOneTime:   1,
		},
		{
			name: "hidden excluded",
			points: []monster2.SpawnPoint{
				{Id: 1, MobTime: 0, Hide: true},
			},
			wantRecurring: 0,
			wantOneTime:   0,
		},
		{
			name:          "empty map",
			points:        nil,
			wantRecurring: 0,
			wantOneTime:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := setupSpawnTestRedis(t)
			r := newTestRegistry(client)
			ctx := context.Background()

			te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
			if err != nil {
				t.Fatalf("tenant.Create: %v", err)
			}
			f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()
			mapKey := character.MapKey{Tenant: te, Field: f}
			mockDP := &mockSpawnDataProcessor{spawnPoints: tc.points}

			if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
				t.Fatalf("InitializeForMap: %v", err)
			}

			recCount, err := r.Count(ctx, mapKey)
			if err != nil {
				t.Fatalf("Count: %v", err)
			}
			if recCount != tc.wantRecurring {
				t.Errorf("Count() = %d, want %d", recCount, tc.wantRecurring)
			}

			oneTimeCount, err := r.CountOneTime(ctx, mapKey)
			if err != nil {
				t.Fatalf("CountOneTime: %v", err)
			}
			if oneTimeCount != tc.wantOneTime {
				t.Errorf("CountOneTime() = %d, want %d", oneTimeCount, tc.wantOneTime)
			}

			seeded, err := r.meta.Get(ctx, mapKey.Tenant, mapKey, metaFieldSeeded)
			if err != nil {
				t.Fatalf("HGet seeded: %v", err)
			}
			if seeded != "1" {
				t.Errorf("seeded = %q, want \"1\"", seeded)
			}
		})
	}
}

func TestInitializeForMap_IsIdempotent(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	ctx := context.Background()

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	mockDP := &mockSpawnDataProcessor{spawnPoints: []monster2.SpawnPoint{
		{Id: 1, MobTime: 0, Hide: false},
		{Id: 2, MobTime: -1, Hide: false},
		{Id: 3, MobTime: 30, Hide: false},
	}}

	if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
		t.Fatalf("InitializeForMap (first): %v", err)
	}

	mockDP.spawnPoints = []monster2.SpawnPoint{
		{Id: 10, MobTime: 0, Hide: false},
		{Id: 11, MobTime: 0, Hide: false},
		{Id: 12, MobTime: 0, Hide: false},
		{Id: 13, MobTime: 0, Hide: false},
		{Id: 14, MobTime: 0, Hide: false},
	}

	if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
		t.Fatalf("InitializeForMap (second): %v", err)
	}

	recCount, err := r.Count(ctx, mapKey)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if recCount != 2 {
		t.Errorf("Count() = %d, want 2 (re-seed must be a no-op)", recCount)
	}

	oneTimeCount, err := r.CountOneTime(ctx, mapKey)
	if err != nil {
		t.Fatalf("CountOneTime: %v", err)
	}
	if oneTimeCount != 1 {
		t.Errorf("CountOneTime() = %d, want 1 (re-seed must be a no-op)", oneTimeCount)
	}
}

func TestRegistryKeys_AreV2AndDistinct(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)

	tid := uuid.New()
	te, err := tenant.Create(tid, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(920010920)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	tests := []struct {
		name       string
		got        string
		wantSuffix string
	}{
		{
			name:       "recurringKey",
			got:        r.recurringKey(mapKey),
			wantSuffix: ":v2:1:2:920010920:00000000-0000-0000-0000-000000000000",
		},
		{
			name:       "oneTimeKey",
			got:        r.oneTimeKey(mapKey),
			wantSuffix: ":v2:onetime:1:2:920010920:00000000-0000-0000-0000-000000000000",
		},
		{
			name:       "metaKey",
			got:        r.metaKey(mapKey),
			wantSuffix: ":v2:meta:1:2:920010920:00000000-0000-0000-0000-000000000000",
		},
	}

	seen := map[string]string{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasSuffix(tt.got, tt.wantSuffix) {
				t.Errorf("%s = %q, want suffix %q", tt.name, tt.got, tt.wantSuffix)
			}
			if !strings.Contains(tt.got, ":maps:spawn:") {
				t.Errorf("%s %q does not contain :maps:spawn:", tt.name, tt.got)
			}
			if !strings.Contains(tt.got, tid.String()) {
				t.Errorf("%s %q does not contain tenant id %s", tt.name, tt.got, tid.String())
			}
		})
		seen[tt.name] = tt.got
	}

	if seen["recurringKey"] == seen["oneTimeKey"] || seen["recurringKey"] == seen["metaKey"] || seen["oneTimeKey"] == seen["metaKey"] {
		t.Errorf("keys are not pairwise distinct: recurring=%q oneTime=%q meta=%q", seen["recurringKey"], seen["oneTimeKey"], seen["metaKey"])
	}
}

func TestFlushTenant_ClearsAllThreeHashes(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	ctx := context.Background()
	l := logrus.New()

	tid := uuid.New()
	te, err := tenant.Create(tid, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	mockDP := &mockSpawnDataProcessor{spawnPoints: []monster2.SpawnPoint{
		{Id: 1, MobTime: 0, Hide: false},
		{Id: 2, MobTime: -1, Hide: false},
		{Id: 3, MobTime: 30, Hide: false},
	}}

	if err := r.InitializeForMap(ctx, mapKey, mockDP, l); err != nil {
		t.Fatalf("InitializeForMap: %v", err)
	}

	hashesUnderTest := []*atlasredis.TenantKeyedHash[character.MapKey]{r.hashes, r.oneTime, r.meta}
	for _, h := range hashesUnderTest {
		entries, err := h.GetAll(ctx, mapKey.Tenant, mapKey)
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(entries) == 0 {
			t.Fatalf("hash should exist before flush")
		}
	}

	deleted, err := r.FlushTenant(ctx, l, tid)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}

	for _, h := range hashesUnderTest {
		entries, err := h.GetAll(ctx, mapKey.Tenant, mapKey)
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("hash should not exist after flush")
		}
	}
}

func newClaimTestMapKey(t *testing.T) character.MapKey {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()
	return character.MapKey{Tenant: te, Field: f}
}

func TestClaimOneTimeSpawnPoints(t *testing.T) {
	tests := []struct {
		name    string
		points  []monster2.SpawnPoint
		wantLen int
		// checkNextSpawnAtPreserved additionally snapshots each recurring
		// point's NextSpawnAt before the claim and asserts it is unchanged
		// after, since only the recurring-only case exercises that guarantee.
		checkNextSpawnAtPreserved bool
		// assert runs after InitializeForMap and the first claim, with the
		// first claim's result. It carries each case's assertions that don't
		// fit the shared before/after-claim skeleton.
		assert func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey, claimBefore time.Time, claimed []*CooldownSpawnPoint)
	}{
		{
			name: "armed field fires the full batch",
			points: func() []monster2.SpawnPoint {
				var pts []monster2.SpawnPoint
				for i := uint32(1); i <= 10; i++ {
					pts = append(pts, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
				}
				return pts
			}(),
			wantLen: 10,
			assert: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey, before time.Time, claimed []*CooldownSpawnPoint) {
				gotIds := map[uint32]bool{}
				for _, cp := range claimed {
					gotIds[cp.Id] = true
					if cp.Template != 9300044 {
						t.Errorf("claimed point %d Template = %d, want 9300044", cp.Id, cp.Template)
					}
				}
				for i := uint32(1); i <= 10; i++ {
					if !gotIds[i] {
						t.Errorf("claimed set missing id %d", i)
					}
				}

				firedStr, err := r.meta.Get(ctx, mapKey.Tenant, mapKey, metaFieldOneTimeFired)
				if err != nil {
					t.Fatalf("HGet onetimeFired: %v", err)
				}
				firedMillis, err := strconv.ParseInt(firedStr, 10, 64)
				if err != nil {
					t.Fatalf("onetimeFired %q not parseable as int64: %v", firedStr, err)
				}
				firedAt := time.UnixMilli(firedMillis)
				if firedAt.Before(before.Add(-60*time.Second)) || firedAt.After(time.Now().Add(60*time.Second)) {
					t.Errorf("onetimeFired timestamp %v not within 60s of now", firedAt)
				}

				hlen, err := r.oneTime.Len(ctx, mapKey.Tenant, mapKey)
				if err != nil {
					t.Fatalf("HLen one-time hash: %v", err)
				}
				if hlen != 10 {
					t.Errorf("one-time hash HLEN = %d, want 10 (claim must not consume the points)", hlen)
				}
			},
		},
		{
			name: "recurring-only field claims nothing",
			points: func() []monster2.SpawnPoint {
				var pts []monster2.SpawnPoint
				for i := uint32(1); i <= 4; i++ {
					pts = append(pts, monster2.SpawnPoint{Id: i, MobTime: 0, Hide: false})
				}
				return pts
			}(),
			wantLen:                   0,
			checkNextSpawnAtPreserved: true,
			assert: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey, before time.Time, claimed []*CooldownSpawnPoint) {
				count, err := r.Count(ctx, mapKey)
				if err != nil {
					t.Fatalf("Count: %v", err)
				}
				if count != 4 {
					t.Errorf("Count() = %d, want 4", count)
				}
			},
		},
		{
			name: "mixed field fires only the one-time subset",
			points: []monster2.SpawnPoint{
				{Id: 1, MobTime: 0, Hide: false},
				{Id: 2, MobTime: -1, Hide: false},
				{Id: 3, MobTime: 30, Hide: false},
			},
			wantLen: 1,
			assert: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey, before time.Time, claimed []*CooldownSpawnPoint) {
				if claimed[0].Id != 2 {
					t.Errorf("claimed point Id = %d, want 2", claimed[0].Id)
				}
			},
		},
		{
			name:    "unseeded field returns nothing",
			points:  nil,
			wantLen: 0,
			assert: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey, before time.Time, claimed []*CooldownSpawnPoint) {
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := setupSpawnTestRedis(t)
			r := newTestRegistry(client)
			ctx := context.Background()
			mapKey := newClaimTestMapKey(t)

			if tc.points != nil {
				mockDP := &mockSpawnDataProcessor{spawnPoints: tc.points}
				if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
					t.Fatalf("InitializeForMap: %v", err)
				}
			}

			var beforeNextSpawnAt map[uint32]time.Time
			if tc.checkNextSpawnAtPreserved {
				before, ok := r.GetSpawnPointsForMap(ctx, mapKey)
				if !ok {
					t.Fatalf("GetSpawnPointsForMap before claim returned not-ok")
				}
				beforeNextSpawnAt = map[uint32]time.Time{}
				for _, sp := range before {
					beforeNextSpawnAt[sp.Id] = sp.NextSpawnAt
				}
			}

			before := time.Now()
			claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
			if err != nil {
				t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
			}
			if len(claimed) != tc.wantLen {
				t.Fatalf("first claim len = %d, want %d", len(claimed), tc.wantLen)
			}

			tc.assert(t, r, ctx, mapKey, before, claimed)

			if beforeNextSpawnAt != nil {
				after, ok := r.GetSpawnPointsForMap(ctx, mapKey)
				if !ok {
					t.Fatalf("GetSpawnPointsForMap after claim returned not-ok")
				}
				for _, sp := range after {
					if !sp.NextSpawnAt.Equal(beforeNextSpawnAt[sp.Id]) {
						t.Errorf("recurring point %d NextSpawnAt changed: before %v, after %v", sp.Id, beforeNextSpawnAt[sp.Id], sp.NextSpawnAt)
					}
				}
			}

			claimed2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
			if err != nil {
				t.Fatalf("ClaimOneTimeSpawnPoints (second): %v", err)
			}
			if len(claimed2) != 0 {
				t.Errorf("second claim len = %d, want 0", len(claimed2))
			}
		})
	}
}

// TestClaimOneTimeSpawnPoints_ConcurrentFiresExactlyOnce is the FR-2.3
// concurrency guarantee: no matter how many callers race, exactly one gets the
// non-empty batch.
func TestClaimOneTimeSpawnPoints_ConcurrentFiresExactlyOnce(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	ctx := context.Background()
	mapKey := newClaimTestMapKey(t)

	var points []monster2.SpawnPoint
	for i := uint32(1); i <= 10; i++ {
		points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
	}
	mockDP := &mockSpawnDataProcessor{spawnPoints: points}
	if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
		t.Fatalf("InitializeForMap: %v", err)
	}

	var mu sync.Mutex
	var lens []int
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
			if err != nil {
				t.Errorf("ClaimOneTimeSpawnPoints: %v", err)
				return
			}
			mu.Lock()
			lens = append(lens, len(claimed))
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(lens) != 8 {
		t.Fatalf("got %d results, want 8", len(lens))
	}
	fullBatches, empties := 0, 0
	for _, l := range lens {
		switch l {
		case 10:
			fullBatches++
		case 0:
			empties++
		default:
			t.Errorf("unexpected claim length %d", l)
		}
	}
	if fullBatches != 1 {
		t.Errorf("fullBatches = %d, want 1", fullBatches)
	}
	if empties != 7 {
		t.Errorf("empties = %d, want 7", empties)
	}
}

func TestRearmOneTime(t *testing.T) {
	// Each scenario needs a distinct sequence of registry calls and
	// assertions (fire-then-rearm-twice, never-fired, rearm-then-reclaim,
	// rearm-leaves-recurring-untouched), so the shared skeleton is only the
	// fresh redis+registry+mapKey fixture; the rest is per-case wiring.
	tests := []struct {
		name string
		run  func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey)
	}{
		{
			name: "fired field re-arms once",
			run: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey) {
				var points []monster2.SpawnPoint
				for i := uint32(1); i <= 10; i++ {
					points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
				}
				mockDP := &mockSpawnDataProcessor{spawnPoints: points}
				if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
					t.Fatalf("InitializeForMap: %v", err)
				}
				if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints: %v", err)
				}

				rearmed, err := r.RearmOneTime(ctx, mapKey)
				if err != nil {
					t.Fatalf("RearmOneTime (first): %v", err)
				}
				if !rearmed {
					t.Errorf("first RearmOneTime = false, want true")
				}

				rearmed2, err := r.RearmOneTime(ctx, mapKey)
				if err != nil {
					t.Fatalf("RearmOneTime (second): %v", err)
				}
				if rearmed2 {
					t.Errorf("second RearmOneTime = true, want false")
				}
			},
		},
		{
			name: "never-fired field returns false",
			run: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey) {
				var points []monster2.SpawnPoint
				for i := uint32(1); i <= 10; i++ {
					points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
				}
				mockDP := &mockSpawnDataProcessor{spawnPoints: points}
				if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
					t.Fatalf("InitializeForMap: %v", err)
				}

				rearmed, err := r.RearmOneTime(ctx, mapKey)
				if err != nil {
					t.Fatalf("RearmOneTime: %v", err)
				}
				if rearmed {
					t.Errorf("RearmOneTime = true, want false")
				}
			},
		},
		{
			name: "re-armed field fires a fresh full batch",
			run: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey) {
				var points []monster2.SpawnPoint
				for i := uint32(1); i <= 10; i++ {
					points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
				}
				mockDP := &mockSpawnDataProcessor{spawnPoints: points}
				if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
					t.Fatalf("InitializeForMap: %v", err)
				}

				claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
				if err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
				}
				if len(claimed) != 10 {
					t.Fatalf("first claim len = %d, want 10", len(claimed))
				}

				rearmed, err := r.RearmOneTime(ctx, mapKey)
				if err != nil {
					t.Fatalf("RearmOneTime: %v", err)
				}
				if !rearmed {
					t.Fatalf("RearmOneTime = false, want true")
				}

				claimed2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
				if err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints (second): %v", err)
				}
				if len(claimed2) != 10 {
					t.Errorf("second claim len = %d, want 10", len(claimed2))
				}
			},
		},
		{
			name: "re-arm leaves the recurring hash untouched",
			run: func(t *testing.T, r *SpawnPointRegistry, ctx context.Context, mapKey character.MapKey) {
				mockDP := &mockSpawnDataProcessor{spawnPoints: []monster2.SpawnPoint{
					{Id: 1, MobTime: 30, Hide: false},
					{Id: 2, MobTime: 30, Hide: false},
					{Id: 3, MobTime: -1, Hide: false},
				}}
				if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
					t.Fatalf("InitializeForMap: %v", err)
				}

				before, ok := r.GetSpawnPointsForMap(ctx, mapKey)
				if !ok {
					t.Fatalf("GetSpawnPointsForMap before claim returned not-ok")
				}
				beforeNextSpawnAt := map[uint32]time.Time{}
				for _, sp := range before {
					beforeNextSpawnAt[sp.Id] = sp.NextSpawnAt
				}

				if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
					t.Fatalf("ClaimOneTimeSpawnPoints: %v", err)
				}
				if _, err := r.RearmOneTime(ctx, mapKey); err != nil {
					t.Fatalf("RearmOneTime: %v", err)
				}

				hlen, err := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
				if err != nil {
					t.Fatalf("HLen recurring hash: %v", err)
				}
				if hlen != 2 {
					t.Errorf("recurring hash HLEN = %d, want 2", hlen)
				}

				after, ok := r.GetSpawnPointsForMap(ctx, mapKey)
				if !ok {
					t.Fatalf("GetSpawnPointsForMap after re-arm returned not-ok")
				}
				if len(after) != 2 {
					t.Fatalf("recurring points after re-arm = %d, want 2", len(after))
				}
				for _, sp := range after {
					if !sp.NextSpawnAt.Equal(beforeNextSpawnAt[sp.Id]) {
						t.Errorf("recurring point %d NextSpawnAt changed: before %v, after %v", sp.Id, beforeNextSpawnAt[sp.Id], sp.NextSpawnAt)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := setupSpawnTestRedis(t)
			r := newTestRegistry(client)
			ctx := context.Background()
			mapKey := newClaimTestMapKey(t)
			tc.run(t, r, ctx, mapKey)
		})
	}
}

// TestRearmOneTime_ConcurrentTrueExactlyOnce pins RearmOneTime's atomicity: no
// matter how many callers race against the same fired field, exactly one may
// observe true. map/processor.go's Exit uses that bool as the exactly-once
// gate for its DESTROY_FIELD emit (design D7), so two callers both observing
// true would double-emit DESTROY_FIELD for a single re-arm.
func TestRearmOneTime_ConcurrentTrueExactlyOnce(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	ctx := context.Background()
	mapKey := newClaimTestMapKey(t)

	var points []monster2.SpawnPoint
	for i := uint32(1); i <= 10; i++ {
		points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
	}
	mockDP := &mockSpawnDataProcessor{spawnPoints: points}
	if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
		t.Fatalf("InitializeForMap: %v", err)
	}
	if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey); err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints: %v", err)
	}

	var mu sync.Mutex
	var results []bool
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rearmed, err := r.RearmOneTime(ctx, mapKey)
			if err != nil {
				t.Errorf("RearmOneTime: %v", err)
				return
			}
			mu.Lock()
			results = append(results, rearmed)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(results) != 8 {
		t.Fatalf("got %d results, want 8", len(results))
	}
	trues := 0
	for _, v := range results {
		if v {
			trues++
		}
	}
	if trues != 1 {
		t.Errorf("RearmOneTime returned true %d times across 8 concurrent callers, want exactly 1", trues)
	}
}

// TestRearmOneTime_IsPerFieldKey verifies re-arming one field's one-time state
// is fully scoped by the field key: channel and instance are both part of the
// key, so re-arming one field must never affect another.
func TestRearmOneTime_IsPerFieldKey(t *testing.T) {
	tests := []struct {
		name   string
		fieldA field.Model
		fieldB field.Model
		labelA string
		labelB string
	}{
		{
			name:   "per channel",
			fieldA: field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build(),
			fieldB: field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(920010920)).Build(),
			labelA: "ch0",
			labelB: "ch1",
		},
		{
			name:   "per instance",
			fieldA: field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).SetInstance(uuid.New()).Build(),
			fieldB: field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).SetInstance(uuid.New()).Build(),
			labelA: "inst1",
			labelB: "inst2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := setupSpawnTestRedis(t)
			r := newTestRegistry(client)
			ctx := context.Background()
			te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
			if err != nil {
				t.Fatalf("tenant.Create: %v", err)
			}

			var points []monster2.SpawnPoint
			for i := uint32(1); i <= 10; i++ {
				points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
			}
			mockDP := &mockSpawnDataProcessor{spawnPoints: points}

			mapKeyA := character.MapKey{Tenant: te, Field: tc.fieldA}
			mapKeyB := character.MapKey{Tenant: te, Field: tc.fieldB}

			if err := r.InitializeForMap(ctx, mapKeyA, mockDP, logrus.New()); err != nil {
				t.Fatalf("InitializeForMap %s: %v", tc.labelA, err)
			}
			if err := r.InitializeForMap(ctx, mapKeyB, mockDP, logrus.New()); err != nil {
				t.Fatalf("InitializeForMap %s: %v", tc.labelB, err)
			}

			if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyA); err != nil {
				t.Fatalf("claim %s: %v", tc.labelA, err)
			}
			if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyB); err != nil {
				t.Fatalf("claim %s: %v", tc.labelB, err)
			}

			if _, err := r.RearmOneTime(ctx, mapKeyA); err != nil {
				t.Fatalf("RearmOneTime %s: %v", tc.labelA, err)
			}

			claimedA, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyA)
			if err != nil {
				t.Fatalf("claim %s after rearm: %v", tc.labelA, err)
			}
			if len(claimedA) != 10 {
				t.Errorf("%s claim after rearm = %d, want 10", tc.labelA, len(claimedA))
			}

			claimedB, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyB)
			if err != nil {
				t.Fatalf("claim %s after %s rearm: %v", tc.labelB, tc.labelA, err)
			}
			if len(claimedB) != 0 {
				t.Errorf("%s claim after %s rearm = %d, want 0", tc.labelB, tc.labelA, len(claimedB))
			}
		})
	}
}

// TestFlushTenant_ReArmsDisarmedField confirms FlushTenant's namespace-wide
// SCAN clears the meta hash along with recurring/one-time, so a field that had
// fired reseeds and re-arms cleanly on the next InitializeForMap.
func TestFlushTenant_ReArmsDisarmedField(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	ctx := context.Background()
	l := logrus.New()
	mapKey := newClaimTestMapKey(t)
	tid := mapKey.Tenant.Id()

	var points []monster2.SpawnPoint
	for i := uint32(1); i <= 10; i++ {
		points = append(points, monster2.SpawnPoint{Id: i, Template: 9300044, MobTime: -1, Hide: false})
	}
	mockDP := &mockSpawnDataProcessor{spawnPoints: points}
	if err := r.InitializeForMap(ctx, mapKey, mockDP, l); err != nil {
		t.Fatalf("InitializeForMap: %v", err)
	}

	claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
	}
	if len(claimed) != 10 {
		t.Fatalf("first claim len = %d, want 10", len(claimed))
	}

	if _, err := r.FlushTenant(ctx, l, tid); err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}

	if err := r.InitializeForMap(ctx, mapKey, mockDP, l); err != nil {
		t.Fatalf("InitializeForMap (post-flush): %v", err)
	}

	claimed2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("ClaimOneTimeSpawnPoints (post-flush): %v", err)
	}
	if len(claimed2) != 10 {
		t.Errorf("post-flush claim len = %d, want 10", len(claimed2))
	}
}
