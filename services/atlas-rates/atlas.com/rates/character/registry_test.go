package character

import (
	"atlas-rates/rate"
	"context"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func setupTestRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
	InitItemTracker(client)
	InitInitializedRegistry(client)
}

func createTestTenantForRegistry() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func createTestCtx(ten tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), ten)
}

func TestRegistryGet_NotFound(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)

	_, err := GetRegistry().Get(ctx, 12345)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestRegistryGetOrCreate_Creates(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)

	ch := channel.NewModel(1, 2)
	m := GetRegistry().GetOrCreate(ctx, ch, 12345)

	if m.CharacterId() != 12345 {
		t.Errorf("CharacterId() = %v, want 12345", m.CharacterId())
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId() = %v, want 1", m.WorldId())
	}
	if m.ChannelId() != 2 {
		t.Errorf("ChannelId() = %v, want 2", m.ChannelId())
	}
}

func TestRegistryGetOrCreate_ReturnsExisting(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	// Create first
	m1 := GetRegistry().GetOrCreate(ctx, ch, 12345)

	// Add a factor to identify it
	f := rate.NewFactor("test", rate.TypeExp, 2.0)
	m1 = m1.WithFactor(f)
	GetRegistry().Update(ctx, m1)

	// Get again
	m2 := GetRegistry().GetOrCreate(ctx, ch, 12345)

	if len(m2.Factors()) != 1 {
		t.Errorf("GetOrCreate() did not return existing model")
	}
}

func TestRegistryUpdate(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	m := GetRegistry().GetOrCreate(ctx, ch, 12345)
	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	GetRegistry().Update(ctx, m)

	retrieved, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", retrieved.ComputedRates().ExpRate())
	}
}

func TestRegistryAddFactor_CreatesIfNotExists(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m := GetRegistry().AddFactor(ctx, ch, 12345, f)

	if m.CharacterId() != 12345 {
		t.Errorf("CharacterId() = %v, want 12345", m.CharacterId())
	}
	if m.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", m.ComputedRates().ExpRate())
	}
}

func TestRegistryAddFactor_UpdatesExisting(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	GetRegistry().AddFactor(ctx, ch, 12345, f1)

	f2 := rate.NewFactor("buff:123", rate.TypeExp, 1.5)
	m := GetRegistry().AddFactor(ctx, ch, 12345, f2)

	// 2.0 * 1.5 = 3.0
	if m.ComputedRates().ExpRate() != 3.0 {
		t.Errorf("ExpRate() = %v, want 3.0", m.ComputedRates().ExpRate())
	}
}

func TestRegistryRemoveFactor_Success(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	GetRegistry().AddFactor(ctx, ch, 12345, f)

	m, err := GetRegistry().RemoveFactor(ctx, 12345, "world", rate.TypeExp)
	if err != nil {
		t.Fatalf("RemoveFactor() error = %v", err)
	}

	if m.ComputedRates().ExpRate() != 1.0 {
		t.Errorf("ExpRate() = %v, want 1.0", m.ComputedRates().ExpRate())
	}
}

func TestRegistryRemoveFactor_NotFound(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)

	_, err := GetRegistry().RemoveFactor(ctx, 99999, "world", rate.TypeExp)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveFactor() error = %v, want ErrNotFound", err)
	}
}

func TestRegistryRemoveFactorsBySource(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	f2 := rate.NewFactor("world", rate.TypeMeso, 1.5)
	f3 := rate.NewFactor("buff:123", rate.TypeExp, 1.2)

	GetRegistry().AddFactor(ctx, ch, 12345, f1)
	GetRegistry().AddFactor(ctx, ch, 12345, f2)
	GetRegistry().AddFactor(ctx, ch, 12345, f3)

	m, err := GetRegistry().RemoveFactorsBySource(ctx, 12345, "world")
	if err != nil {
		t.Fatalf("RemoveFactorsBySource() error = %v", err)
	}

	if len(m.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m.Factors()))
	}
	if m.Factors()[0].Source() != "buff:123" {
		t.Errorf("Remaining factor source = %v, want buff:123", m.Factors()[0].Source())
	}
}

func TestRegistryGetAllForWorld(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch1 := channel.NewModel(1, 1)
	ch2 := channel.NewModel(1, 2)
	ch3 := channel.NewModel(2, 1)

	// Create characters in different worlds
	GetRegistry().GetOrCreate(ctx, ch1, 100)
	GetRegistry().GetOrCreate(ctx, ch2, 101)
	GetRegistry().GetOrCreate(ctx, ch3, 200)

	world1Chars := GetRegistry().GetAllForWorld(ctx, 1)
	if len(world1Chars) != 2 {
		t.Errorf("GetAllForWorld(1) returned %v characters, want 2", len(world1Chars))
	}

	world2Chars := GetRegistry().GetAllForWorld(ctx, 2)
	if len(world2Chars) != 1 {
		t.Errorf("GetAllForWorld(2) returned %v characters, want 1", len(world2Chars))
	}
}

func TestRegistryUpdateWorldRate(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch1 := channel.NewModel(1, 1)
	ch2 := channel.NewModel(1, 2)
	ch3 := channel.NewModel(2, 1)

	// Create characters in different worlds
	GetRegistry().GetOrCreate(ctx, ch1, 100)
	GetRegistry().GetOrCreate(ctx, ch2, 101)
	GetRegistry().GetOrCreate(ctx, ch3, 200)

	// Update world 1 rate
	GetRegistry().UpdateWorldRate(ctx, 1, rate.TypeExp, 2.0)

	// Check world 1 characters
	m100, _ := GetRegistry().Get(ctx, 100)
	m101, _ := GetRegistry().Get(ctx, 101)
	m200, _ := GetRegistry().Get(ctx, 200)

	if m100.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("Character 100 ExpRate() = %v, want 2.0", m100.ComputedRates().ExpRate())
	}
	if m101.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("Character 101 ExpRate() = %v, want 2.0", m101.ComputedRates().ExpRate())
	}
	// World 2 character should be unaffected
	if m200.ComputedRates().ExpRate() != 1.0 {
		t.Errorf("Character 200 ExpRate() = %v, want 1.0", m200.ComputedRates().ExpRate())
	}
}

func TestRegistryDelete(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForRegistry()
	ctx := createTestCtx(ten)
	ch := channel.NewModel(1, 2)

	GetRegistry().GetOrCreate(ctx, ch, 12345)

	GetRegistry().Delete(ctx, 12345)

	_, err := GetRegistry().Get(ctx, 12345)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestRegistryTenantIsolation(t *testing.T) {
	setupTestRegistries(t)

	t1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	t2, _ := tenant.Create(uuid.New(), "KMS", 1, 2)
	ctx1 := createTestCtx(t1)
	ctx2 := createTestCtx(t2)
	ch := channel.NewModel(1, 1)

	// Add same character ID in different tenants
	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	f2 := rate.NewFactor("world", rate.TypeExp, 3.0)

	GetRegistry().AddFactor(ctx1, ch, 12345, f1)
	GetRegistry().AddFactor(ctx2, ch, 12345, f2)

	m1, _ := GetRegistry().Get(ctx1, 12345)
	m2, _ := GetRegistry().Get(ctx2, 12345)

	if m1.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("Tenant 1 ExpRate() = %v, want 2.0", m1.ComputedRates().ExpRate())
	}
	if m2.ComputedRates().ExpRate() != 3.0 {
		t.Errorf("Tenant 2 ExpRate() = %v, want 3.0", m2.ComputedRates().ExpRate())
	}
}
