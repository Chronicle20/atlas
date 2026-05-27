package monster

import (
	"context"
	"fmt"
	"testing"
	"time"

	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func setupSpawnTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), mr
}

// newTestRegistry builds a fully-initialized SpawnPointRegistry suitable for
// use in tests. Using this instead of &SpawnPointRegistry{client: client}
// ensures the hashes field is wired up correctly.
func newTestRegistry(client *goredis.Client) *SpawnPointRegistry {
	kh := atlasredis.NewKeyedHash[character.MapKey](client, "maps:spawn", func(mk character.MapKey) string {
		return fmt.Sprintf("%s:%d:%d:%d:%s",
			mk.Tenant.Id().String(),
			mk.Field.WorldId(),
			mk.Field.ChannelId(),
			mk.Field.MapId(),
			mk.Field.Instance().String(),
		)
	})
	return &SpawnPointRegistry{client: client, hashes: kh}
}

func TestSpawnPointRegistry_FlushTenant_DeletesAllForTenant(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := newTestRegistry(client)
	tid := uuid.New()
	ctx := context.Background()
	l := logrus.New()

	for i := 0; i < 3; i++ {
		k := fmt.Sprintf("atlas:maps:spawn:%s:0:0:%d:00000000-0000-0000-0000-000000000000", tid.String(), 100+i)
		if err := client.HSet(ctx, k, "1", "{}").Err(); err != nil {
			t.Fatalf("HSet seed: %v", err)
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

	_ = client.HSet(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tA), "1", "{}").Err()
	_ = client.HSet(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tB), "1", "{}").Err()

	deleted, err := r.FlushTenant(ctx, l, tA)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	n, _ := client.Exists(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tB)).Result()
	if n != 1 {
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
