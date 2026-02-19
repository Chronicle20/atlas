package shops

import (
	"context"
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

func testTenantCtx() context.Context {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), ten)
}

func TestAddAndGetShop(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	GetRegistry().AddCharacter(ctx, 100, 5000)

	shopId, ok := GetRegistry().GetShop(ctx, 100)
	if !ok {
		t.Fatal("Expected character to be in shop")
	}
	if shopId != 5000 {
		t.Errorf("Expected shopId 5000, got %d", shopId)
	}
}

func TestGetShopNotFound(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	_, ok := GetRegistry().GetShop(ctx, 999)
	if ok {
		t.Error("Expected character to not be in shop")
	}
}

func TestRemoveCharacter(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	GetRegistry().AddCharacter(ctx, 100, 5000)
	GetRegistry().RemoveCharacter(ctx, 100)

	_, ok := GetRegistry().GetShop(ctx, 100)
	if ok {
		t.Error("Expected character to not be in shop after removal")
	}
}

func TestAddCharacterSwitchShop(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	GetRegistry().AddCharacter(ctx, 100, 5000)
	GetRegistry().AddCharacter(ctx, 100, 6000)

	shopId, ok := GetRegistry().GetShop(ctx, 100)
	if !ok {
		t.Fatal("Expected character to be in shop")
	}
	if shopId != 6000 {
		t.Errorf("Expected shopId 6000, got %d", shopId)
	}

	chars := GetRegistry().GetCharactersInShop(ctx, 5000)
	if len(chars) != 0 {
		t.Errorf("Expected 0 characters in old shop, got %d", len(chars))
	}

	chars = GetRegistry().GetCharactersInShop(ctx, 6000)
	if len(chars) != 1 || chars[0] != 100 {
		t.Errorf("Expected [100] in new shop, got %v", chars)
	}
}

func TestGetCharactersInShop(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	GetRegistry().AddCharacter(ctx, 100, 5000)
	GetRegistry().AddCharacter(ctx, 200, 5000)
	GetRegistry().AddCharacter(ctx, 300, 6000)

	chars := GetRegistry().GetCharactersInShop(ctx, 5000)
	if len(chars) != 2 {
		t.Fatalf("Expected 2 characters in shop 5000, got %d", len(chars))
	}

	charSet := make(map[uint32]bool)
	for _, c := range chars {
		charSet[c] = true
	}
	if !charSet[100] || !charSet[200] {
		t.Errorf("Expected characters 100 and 200 in shop, got %v", chars)
	}

	chars = GetRegistry().GetCharactersInShop(ctx, 6000)
	if len(chars) != 1 || chars[0] != 300 {
		t.Errorf("Expected [300] in shop 6000, got %v", chars)
	}
}

func TestGetCharactersInShopEmpty(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	chars := GetRegistry().GetCharactersInShop(ctx, 9999)
	if len(chars) != 0 {
		t.Errorf("Expected 0 characters, got %d", len(chars))
	}
}

func TestRemoveCharacterUpdatesShopSet(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	GetRegistry().AddCharacter(ctx, 100, 5000)
	GetRegistry().AddCharacter(ctx, 200, 5000)
	GetRegistry().RemoveCharacter(ctx, 100)

	chars := GetRegistry().GetCharactersInShop(ctx, 5000)
	if len(chars) != 1 || chars[0] != 200 {
		t.Errorf("Expected [200] after removing 100, got %v", chars)
	}
}

func TestTenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	ctx1 := testTenantCtx()
	ctx2 := testTenantCtx()

	GetRegistry().AddCharacter(ctx1, 100, 5000)
	GetRegistry().AddCharacter(ctx2, 100, 6000)

	shopId1, ok := GetRegistry().GetShop(ctx1, 100)
	if !ok || shopId1 != 5000 {
		t.Errorf("Expected tenant1 character in shop 5000, got %d", shopId1)
	}

	shopId2, ok := GetRegistry().GetShop(ctx2, 100)
	if !ok || shopId2 != 6000 {
		t.Errorf("Expected tenant2 character in shop 6000, got %d", shopId2)
	}
}

func TestRemoveNonExistent(t *testing.T) {
	setupTestRegistry(t)
	ctx := testTenantCtx()

	// Should not panic
	GetRegistry().RemoveCharacter(ctx, 999)
}
