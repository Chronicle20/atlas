package monster

import (
	"context"
	"fmt"
	"testing"

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

func TestSpawnPointRegistry_FlushTenant_DeletesAllForTenant(t *testing.T) {
	client, _ := setupSpawnTestRedis(t)
	r := &SpawnPointRegistry{client: client}
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
	r := &SpawnPointRegistry{client: client}
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
	r := &SpawnPointRegistry{client: client}
	deleted, err := r.FlushTenant(context.Background(), logrus.New(), uuid.New())
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}
