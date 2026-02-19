package chalkboard

import (
	"context"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testCtx() context.Context {
	st, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), st)
}

func TestRegistryGetNonExistent(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	_, ok := getRegistry().Get(ctx, 12345)
	if ok {
		t.Error("Expected Get to return false for non-existent key")
	}
}

func TestRegistrySetAndGet(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()
	characterId := uint32(12345)
	message := "Hello, World!"

	getRegistry().Set(ctx, characterId, message)

	got, ok := getRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected Get to return true for existing key")
	}
	if got != message {
		t.Errorf("Expected message %q, got %q", message, got)
	}
}

func TestRegistryClear(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()
	characterId := uint32(12345)

	getRegistry().Set(ctx, characterId, "test message")

	cleared := getRegistry().Clear(ctx, characterId)
	if !cleared {
		t.Error("Expected Clear to return true for existing key")
	}

	_, ok := getRegistry().Get(ctx, characterId)
	if ok {
		t.Error("Expected Get to return false after Clear")
	}
}

func TestRegistryClearNonExistent(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	cleared := getRegistry().Clear(ctx, 99999)
	if cleared {
		t.Error("Expected Clear to return false for non-existent key")
	}
}

func TestRegistryTenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := tenant.WithContext(context.Background(), tenant1)
	ctx2 := tenant.WithContext(context.Background(), tenant2)
	characterId := uint32(12345)

	getRegistry().Set(ctx1, characterId, "tenant1 message")
	getRegistry().Set(ctx2, characterId, "tenant2 message")

	msg1, ok1 := getRegistry().Get(ctx1, characterId)
	if !ok1 {
		t.Fatal("Expected Get to return true for tenant1")
	}
	if msg1 != "tenant1 message" {
		t.Errorf("Expected tenant1 message, got %q", msg1)
	}

	msg2, ok2 := getRegistry().Get(ctx2, characterId)
	if !ok2 {
		t.Fatal("Expected Get to return true for tenant2")
	}
	if msg2 != "tenant2 message" {
		t.Errorf("Expected tenant2 message, got %q", msg2)
	}

	getRegistry().Clear(ctx1, characterId)

	_, ok1After := getRegistry().Get(ctx1, characterId)
	if ok1After {
		t.Error("Expected tenant1 data to be cleared")
	}

	msg2After, ok2After := getRegistry().Get(ctx2, characterId)
	if !ok2After {
		t.Error("Expected tenant2 data to still exist after clearing tenant1")
	}
	if msg2After != "tenant2 message" {
		t.Errorf("Expected tenant2 message to remain unchanged, got %q", msg2After)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			getRegistry().Set(ctx, characterId, "message")
			getRegistry().Get(ctx, characterId)
			getRegistry().Clear(ctx, characterId)
		}(i)
	}

	wg.Wait()
}

func TestRegistryOverwrite(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()
	characterId := uint32(12345)

	getRegistry().Set(ctx, characterId, "first message")
	getRegistry().Set(ctx, characterId, "second message")

	got, ok := getRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected Get to return true")
	}
	if got != "second message" {
		t.Errorf("Expected overwritten message, got %q", got)
	}
}
