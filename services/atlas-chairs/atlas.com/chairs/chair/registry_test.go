package chair

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func TestRegistry_GetSet(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	characterId := uint32(12345)
	m := Model{id: 1, chairType: "FIXED"}

	_, ok := GetRegistry().Get(ctx, characterId)
	if ok {
		t.Fatal("Expected character to not exist in registry initially")
	}

	GetRegistry().Set(ctx, characterId, m)

	retrieved, ok := GetRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}

	if retrieved.Id() != m.Id() {
		t.Errorf("Id mismatch. Expected %d, got %d", m.Id(), retrieved.Id())
	}

	if retrieved.Type() != m.Type() {
		t.Errorf("Type mismatch. Expected %s, got %s", m.Type(), retrieved.Type())
	}
}

func TestRegistry_Clear(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	characterId := uint32(12345)
	m := Model{id: 1, chairType: "FIXED"}

	GetRegistry().Set(ctx, characterId, m)

	_, ok := GetRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}

	existed := GetRegistry().Clear(ctx, characterId)
	if !existed {
		t.Fatal("Expected Clear to return true for existing entry")
	}

	_, ok = GetRegistry().Get(ctx, characterId)
	if ok {
		t.Fatal("Expected character to not exist in registry after Clear")
	}
}

func TestRegistry_Clear_NotExists(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	characterId := uint32(99999)

	existed := GetRegistry().Clear(ctx, characterId)
	if existed {
		t.Fatal("Expected Clear to return false for non-existent entry")
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			m := Model{id: uint32(id), chairType: "FIXED"}
			GetRegistry().Set(ctx, characterId, m)
		}(i)
	}

	wg.Wait()

	for i := 0; i < iterations; i++ {
		characterId := uint32(i)
		m, ok := GetRegistry().Get(ctx, characterId)
		if !ok {
			t.Errorf("Expected character %d to exist in registry", characterId)
			continue
		}
		if m.Id() != uint32(i) {
			t.Errorf("Expected chair id %d, got %d", i, m.Id())
		}
	}

	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			GetRegistry().Get(ctx, characterId)
		}(i)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			GetRegistry().Clear(ctx, characterId)
		}(i)
	}

	wg.Wait()
}

func TestRegistry_MultipleCharacters(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	chars := []struct {
		characterId uint32
		chairId     uint32
		chairType   string
	}{
		{100, 0, "FIXED"},
		{200, 3010001, "PORTABLE"},
		{300, 1, "FIXED"},
	}

	for _, c := range chars {
		m := Model{id: c.chairId, chairType: c.chairType}
		GetRegistry().Set(ctx, c.characterId, m)
	}

	for _, c := range chars {
		m, ok := GetRegistry().Get(ctx, c.characterId)
		if !ok {
			t.Errorf("Expected character %d to exist", c.characterId)
			continue
		}
		if m.Id() != c.chairId {
			t.Errorf("Character %d: expected chair id %d, got %d", c.characterId, c.chairId, m.Id())
		}
		if m.Type() != c.chairType {
			t.Errorf("Character %d: expected chair type %s, got %s", c.characterId, c.chairType, m.Type())
		}
	}

	GetRegistry().Clear(ctx, 200)

	_, ok := GetRegistry().Get(ctx, 200)
	if ok {
		t.Error("Expected character 200 to be cleared")
	}

	_, ok = GetRegistry().Get(ctx, 100)
	if !ok {
		t.Error("Expected character 100 to still exist")
	}

	_, ok = GetRegistry().Get(ctx, 300)
	if !ok {
		t.Error("Expected character 300 to still exist")
	}
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupTestRegistry(t)

	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := tenant.WithContext(context.Background(), tenant1)
	ctx2 := tenant.WithContext(context.Background(), tenant2)

	characterId := uint32(12345)

	GetRegistry().Set(ctx1, characterId, Model{id: 1, chairType: "FIXED"})
	GetRegistry().Set(ctx2, characterId, Model{id: 2, chairType: "PORTABLE"})

	m1, ok1 := GetRegistry().Get(ctx1, characterId)
	if !ok1 {
		t.Fatal("Expected tenant1 character to exist")
	}
	if m1.Id() != 1 {
		t.Errorf("Tenant1: expected chair id 1, got %d", m1.Id())
	}

	m2, ok2 := GetRegistry().Get(ctx2, characterId)
	if !ok2 {
		t.Fatal("Expected tenant2 character to exist")
	}
	if m2.Id() != 2 {
		t.Errorf("Tenant2: expected chair id 2, got %d", m2.Id())
	}

	GetRegistry().Clear(ctx1, characterId)

	_, ok := GetRegistry().Get(ctx1, characterId)
	if ok {
		t.Error("Expected tenant1 data to be cleared")
	}

	_, ok = GetRegistry().Get(ctx2, characterId)
	if !ok {
		t.Error("Expected tenant2 data to still exist")
	}
}

func TestRegistry_RecoveryTimestampsRoundTrip(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	characterId := uint32(22345)
	m := Model{id: 3010000, chairType: "PORTABLE"}.
		WithHpRecoveryAt(1751234567890).
		WithMpRecoveryAt(1751234570000)

	GetRegistry().Set(ctx, characterId, m)

	got, ok := GetRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}
	if got.LastHpRecoveryAt() != 1751234567890 {
		t.Errorf("lastHpRecoveryAt: got %d, want 1751234567890", got.LastHpRecoveryAt())
	}
	if got.LastMpRecoveryAt() != 1751234570000 {
		t.Errorf("lastMpRecoveryAt: got %d, want 1751234570000", got.LastMpRecoveryAt())
	}
	if got.Id() != 3010000 || got.Type() != "PORTABLE" {
		t.Errorf("existing fields lost in round-trip: id %d type %s", got.Id(), got.Type())
	}

	// A model without timestamps unmarshals to zero (backward-compatible with
	// pre-task-141 registry entries).
	GetRegistry().Set(ctx, characterId+1, Model{id: 1, chairType: "FIXED"})
	got2, _ := GetRegistry().Get(ctx, characterId+1)
	if got2.LastHpRecoveryAt() != 0 || got2.LastMpRecoveryAt() != 0 {
		t.Errorf("expected zero timestamps, got %d/%d", got2.LastHpRecoveryAt(), got2.LastMpRecoveryAt())
	}
}
