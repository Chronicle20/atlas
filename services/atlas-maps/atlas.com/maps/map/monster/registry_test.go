package monster

import (
	"atlas-maps/map/character"
	"context"
	"strings"
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

			seeded, err := client.HGet(ctx, r.metaKey(mapKey), metaFieldSeeded).Result()
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

	keys := []string{r.recurringKey(mapKey), r.oneTimeKey(mapKey), r.metaKey(mapKey)}
	for _, k := range keys {
		n, err := client.Exists(ctx, k).Result()
		if err != nil {
			t.Fatalf("Exists(%q): %v", k, err)
		}
		if n != 1 {
			t.Fatalf("key %q should exist before flush", k)
		}
	}

	deleted, err := r.FlushTenant(ctx, l, tid)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}

	for _, k := range keys {
		n, err := client.Exists(ctx, k).Result()
		if err != nil {
			t.Fatalf("Exists(%q): %v", k, err)
		}
		if n != 0 {
			t.Fatalf("key %q should not exist after flush", k)
		}
	}
}
