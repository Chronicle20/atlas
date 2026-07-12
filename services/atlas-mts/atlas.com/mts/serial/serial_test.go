package serial_test

import (
	"context"
	"testing"

	"atlas-mts/serial"
	"atlas-mts/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func serialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, serial.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

// TestNextIsMonotonicPerWorld asserts the per-(tenant, world) counter starts at
// 1 and increments by 1 on each call.
func TestNextIsMonotonicPerWorld(t *testing.T) {
	tenantId := uuid.New()
	db := serialTestDB(t).WithContext(tenantCtx(t, tenantId))

	for i := uint32(1); i <= 5; i++ {
		got, err := serial.Next(db, tenantId, 0)
		if err != nil {
			t.Fatalf("Next #%d: %v", i, err)
		}
		if got != i {
			t.Errorf("Next #%d = %d, want %d", i, got, i)
		}
	}
}

// TestNextIndependentPerWorld asserts two worlds in the same tenant have
// independent sequences (each starts at 1).
func TestNextIndependentPerWorld(t *testing.T) {
	tenantId := uuid.New()
	db := serialTestDB(t).WithContext(tenantCtx(t, tenantId))

	w0a, err := serial.Next(db, tenantId, world.Id(0))
	if err != nil {
		t.Fatalf("Next world 0: %v", err)
	}
	w1a, err := serial.Next(db, tenantId, world.Id(1))
	if err != nil {
		t.Fatalf("Next world 1: %v", err)
	}
	w0b, err := serial.Next(db, tenantId, world.Id(0))
	if err != nil {
		t.Fatalf("Next world 0 again: %v", err)
	}

	if w0a != 1 {
		t.Errorf("world 0 first = %d, want 1", w0a)
	}
	if w1a != 1 {
		t.Errorf("world 1 first = %d, want 1 (independent sequence)", w1a)
	}
	if w0b != 2 {
		t.Errorf("world 0 second = %d, want 2", w0b)
	}
}

// TestNextIndependentPerTenant asserts two tenants have independent sequences
// for the same world.
func TestNextIndependentPerTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := serialTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	a1, err := serial.Next(dbA, tenantA, world.Id(0))
	if err != nil {
		t.Fatalf("Next tenant A: %v", err)
	}
	b1, err := serial.Next(dbB, tenantB, world.Id(0))
	if err != nil {
		t.Fatalf("Next tenant B: %v", err)
	}
	if a1 != 1 || b1 != 1 {
		t.Errorf("tenant A first=%d, tenant B first=%d; want both 1 (independent)", a1, b1)
	}
}
