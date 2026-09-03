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

func (m *mockSpawnDataProcessor) SpawnableSpawnPointProvider(_ _map.Id) model.Provider[[]monster2.SpawnPoint] {
	return func() ([]monster2.SpawnPoint, error) { return m.spawnPoints, nil }
}

func (m *mockSpawnDataProcessor) GetSpawnPoints(_ _map.Id) ([]monster2.SpawnPoint, error) {
	return m.spawnPoints, nil
}

func (m *mockSpawnDataProcessor) GetSpawnableSpawnPoints(_ _map.Id) ([]monster2.SpawnPoint, error) {
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

	recurring := r.recurringKey(mapKey)
	oneTime := r.oneTimeKey(mapKey)
	meta := r.metaKey(mapKey)

	wantRecSuffix := ":v2:1:2:920010920:00000000-0000-0000-0000-000000000000"
	wantOneSuffix := ":v2:onetime:1:2:920010920:00000000-0000-0000-0000-000000000000"
	wantMetaSuffix := ":v2:meta:1:2:920010920:00000000-0000-0000-0000-000000000000"

	if !strings.HasSuffix(recurring, wantRecSuffix) {
		t.Errorf("recurringKey = %q, want suffix %q", recurring, wantRecSuffix)
	}
	if !strings.HasSuffix(oneTime, wantOneSuffix) {
		t.Errorf("oneTimeKey = %q, want suffix %q", oneTime, wantOneSuffix)
	}
	if !strings.HasSuffix(meta, wantMetaSuffix) {
		t.Errorf("metaKey = %q, want suffix %q", meta, wantMetaSuffix)
	}

	for _, k := range []string{recurring, oneTime, meta} {
		if !strings.Contains(k, ":maps:spawn:") {
			t.Errorf("key %q does not contain :maps:spawn:", k)
		}
		if !strings.Contains(k, tid.String()) {
			t.Errorf("key %q does not contain tenant id %s", k, tid.String())
		}
	}

	if recurring == oneTime || recurring == meta || oneTime == meta {
		t.Errorf("keys are not pairwise distinct: recurring=%q oneTime=%q meta=%q", recurring, oneTime, meta)
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
	t.Run("armed field fires the full batch", func(t *testing.T) {
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

		before := time.Now()
		claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
		}
		if len(claimed) != 10 {
			t.Fatalf("first claim len = %d, want 10", len(claimed))
		}
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

		claimed2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (second): %v", err)
		}
		if len(claimed2) != 0 {
			t.Errorf("second claim len = %d, want 0", len(claimed2))
		}
	})

	t.Run("recurring-only field claims nothing", func(t *testing.T) {
		client, _ := setupSpawnTestRedis(t)
		r := newTestRegistry(client)
		ctx := context.Background()
		mapKey := newClaimTestMapKey(t)

		var points []monster2.SpawnPoint
		for i := uint32(1); i <= 4; i++ {
			points = append(points, monster2.SpawnPoint{Id: i, MobTime: 0, Hide: false})
		}
		mockDP := &mockSpawnDataProcessor{spawnPoints: points}
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

		claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
		}
		if len(claimed) != 0 {
			t.Errorf("first claim len = %d, want 0", len(claimed))
		}

		count, err := r.Count(ctx, mapKey)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if count != 4 {
			t.Errorf("Count() = %d, want 4", count)
		}

		after, ok := r.GetSpawnPointsForMap(ctx, mapKey)
		if !ok {
			t.Fatalf("GetSpawnPointsForMap after claim returned not-ok")
		}
		for _, sp := range after {
			if !sp.NextSpawnAt.Equal(beforeNextSpawnAt[sp.Id]) {
				t.Errorf("recurring point %d NextSpawnAt changed: before %v, after %v", sp.Id, beforeNextSpawnAt[sp.Id], sp.NextSpawnAt)
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

	t.Run("mixed field fires only the one-time subset", func(t *testing.T) {
		client, _ := setupSpawnTestRedis(t)
		r := newTestRegistry(client)
		ctx := context.Background()
		mapKey := newClaimTestMapKey(t)

		mockDP := &mockSpawnDataProcessor{spawnPoints: []monster2.SpawnPoint{
			{Id: 1, MobTime: 0, Hide: false},
			{Id: 2, MobTime: -1, Hide: false},
			{Id: 3, MobTime: 30, Hide: false},
		}}
		if err := r.InitializeForMap(ctx, mapKey, mockDP, logrus.New()); err != nil {
			t.Fatalf("InitializeForMap: %v", err)
		}

		claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
		}
		if len(claimed) != 1 {
			t.Fatalf("first claim len = %d, want 1", len(claimed))
		}
		if claimed[0].Id != 2 {
			t.Errorf("claimed point Id = %d, want 2", claimed[0].Id)
		}

		claimed2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (second): %v", err)
		}
		if len(claimed2) != 0 {
			t.Errorf("second claim len = %d, want 0", len(claimed2))
		}
	})

	t.Run("unseeded field returns nothing", func(t *testing.T) {
		client, _ := setupSpawnTestRedis(t)
		r := newTestRegistry(client)
		ctx := context.Background()
		mapKey := newClaimTestMapKey(t)

		claimed, err := r.ClaimOneTimeSpawnPoints(ctx, mapKey)
		if err != nil {
			t.Fatalf("ClaimOneTimeSpawnPoints (first): %v", err)
		}
		if len(claimed) != 0 {
			t.Errorf("first claim len = %d, want 0", len(claimed))
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
	t.Run("fired field re-arms once", func(t *testing.T) {
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
	})

	t.Run("never-fired field returns false", func(t *testing.T) {
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

		rearmed, err := r.RearmOneTime(ctx, mapKey)
		if err != nil {
			t.Fatalf("RearmOneTime: %v", err)
		}
		if rearmed {
			t.Errorf("RearmOneTime = true, want false")
		}
	})

	t.Run("re-armed field fires a fresh full batch", func(t *testing.T) {
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
	})

	t.Run("re-arm leaves the recurring hash untouched", func(t *testing.T) {
		client, _ := setupSpawnTestRedis(t)
		r := newTestRegistry(client)
		ctx := context.Background()
		mapKey := newClaimTestMapKey(t)

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
	})
}

// TestRearmOneTime_IsPerFieldKey verifies re-arming one field's one-time state
// is fully scoped by the field key: channel and instance are both part of the
// key, so re-arming one field must never affect another.
func TestRearmOneTime_IsPerFieldKey(t *testing.T) {
	t.Run("per channel", func(t *testing.T) {
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

		fCh0 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()
		fCh1 := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(920010920)).Build()
		mapKeyCh0 := character.MapKey{Tenant: te, Field: fCh0}
		mapKeyCh1 := character.MapKey{Tenant: te, Field: fCh1}

		if err := r.InitializeForMap(ctx, mapKeyCh0, mockDP, logrus.New()); err != nil {
			t.Fatalf("InitializeForMap ch0: %v", err)
		}
		if err := r.InitializeForMap(ctx, mapKeyCh1, mockDP, logrus.New()); err != nil {
			t.Fatalf("InitializeForMap ch1: %v", err)
		}

		if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyCh0); err != nil {
			t.Fatalf("claim ch0: %v", err)
		}
		if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyCh1); err != nil {
			t.Fatalf("claim ch1: %v", err)
		}

		if _, err := r.RearmOneTime(ctx, mapKeyCh0); err != nil {
			t.Fatalf("RearmOneTime ch0: %v", err)
		}

		claimedCh0, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyCh0)
		if err != nil {
			t.Fatalf("claim ch0 after rearm: %v", err)
		}
		if len(claimedCh0) != 10 {
			t.Errorf("ch0 claim after rearm = %d, want 10", len(claimedCh0))
		}

		claimedCh1, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyCh1)
		if err != nil {
			t.Fatalf("claim ch1 after ch0 rearm: %v", err)
		}
		if len(claimedCh1) != 0 {
			t.Errorf("ch1 claim after ch0 rearm = %d, want 0", len(claimedCh1))
		}
	})

	t.Run("per instance", func(t *testing.T) {
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

		fInst1 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).SetInstance(uuid.New()).Build()
		fInst2 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).SetInstance(uuid.New()).Build()
		mapKeyInst1 := character.MapKey{Tenant: te, Field: fInst1}
		mapKeyInst2 := character.MapKey{Tenant: te, Field: fInst2}

		if err := r.InitializeForMap(ctx, mapKeyInst1, mockDP, logrus.New()); err != nil {
			t.Fatalf("InitializeForMap inst1: %v", err)
		}
		if err := r.InitializeForMap(ctx, mapKeyInst2, mockDP, logrus.New()); err != nil {
			t.Fatalf("InitializeForMap inst2: %v", err)
		}

		if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyInst1); err != nil {
			t.Fatalf("claim inst1: %v", err)
		}
		if _, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyInst2); err != nil {
			t.Fatalf("claim inst2: %v", err)
		}

		if _, err := r.RearmOneTime(ctx, mapKeyInst1); err != nil {
			t.Fatalf("RearmOneTime inst1: %v", err)
		}

		claimedInst1, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyInst1)
		if err != nil {
			t.Fatalf("claim inst1 after rearm: %v", err)
		}
		if len(claimedInst1) != 10 {
			t.Errorf("inst1 claim after rearm = %d, want 10", len(claimedInst1))
		}

		claimedInst2, err := r.ClaimOneTimeSpawnPoints(ctx, mapKeyInst2)
		if err != nil {
			t.Fatalf("claim inst2 after inst1 rearm: %v", err)
		}
		if len(claimedInst2) != 0 {
			t.Errorf("inst2 claim after inst1 rearm = %d, want 0", len(claimedInst2))
		}
	})
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
