package stackable_test

import (
	"atlas-inventory/stackable"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := stackable.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestCreate(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()
	quantity := uint32(10)
	ownerId := uint32(123)
	flag := uint16(1)
	rechargeable := uint64(100)

	m, err := p.Create(compartmentId, quantity, ownerId, flag, rechargeable)
	if err != nil {
		t.Fatalf("Failed to create stackable: %v", err)
	}

	if m.Id() == 0 {
		t.Error("Expected non-zero Id after create")
	}
	if m.Quantity() != quantity {
		t.Errorf("Expected Quantity %d, got %d", quantity, m.Quantity())
	}
	if m.OwnerId() != ownerId {
		t.Errorf("Expected OwnerId %d, got %d", ownerId, m.OwnerId())
	}
	if m.Flag() != flag {
		t.Errorf("Expected Flag %d, got %d", flag, m.Flag())
	}
	if m.Rechargeable() != rechargeable {
		t.Errorf("Expected Rechargeable %d, got %d", rechargeable, m.Rechargeable())
	}
}

func TestGetById(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()
	created, err := p.Create(compartmentId, 50, 456, 2, 200)
	if err != nil {
		t.Fatalf("Failed to create stackable: %v", err)
	}

	retrieved, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to get stackable by Id: %v", err)
	}

	if retrieved.Id() != created.Id() {
		t.Errorf("Expected Id %d, got %d", created.Id(), retrieved.Id())
	}
	if retrieved.Quantity() != created.Quantity() {
		t.Errorf("Expected Quantity %d, got %d", created.Quantity(), retrieved.Quantity())
	}
	if retrieved.OwnerId() != created.OwnerId() {
		t.Errorf("Expected OwnerId %d, got %d", created.OwnerId(), retrieved.OwnerId())
	}
}

func TestGetByIdNotFound(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	_, err := p.GetById(99999)
	if err == nil {
		t.Error("Expected error when getting non-existent stackable")
	}
}

func TestUpdateQuantity(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()
	created, err := p.Create(compartmentId, 10, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable: %v", err)
	}

	newQuantity := uint32(25)
	err = p.UpdateQuantity(created.Id(), newQuantity)
	if err != nil {
		t.Fatalf("Failed to update quantity: %v", err)
	}

	updated, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to get updated stackable: %v", err)
	}

	if updated.Quantity() != newQuantity {
		t.Errorf("Expected updated Quantity %d, got %d", newQuantity, updated.Quantity())
	}
}

func TestDelete(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()
	created, err := p.Create(compartmentId, 10, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable: %v", err)
	}

	err = p.Delete(created.Id())
	if err != nil {
		t.Fatalf("Failed to delete stackable: %v", err)
	}

	_, err = p.GetById(created.Id())
	if err == nil {
		t.Error("Expected error when getting deleted stackable")
	}
}

func TestByCompartmentIdProvider(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()

	// Create multiple stackables for the same compartment
	_, err := p.Create(compartmentId, 10, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable 1: %v", err)
	}
	_, err = p.Create(compartmentId, 20, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable 2: %v", err)
	}

	// Create a stackable for a different compartment
	otherCompartmentId := uuid.New()
	_, err = p.Create(otherCompartmentId, 30, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable 3: %v", err)
	}

	stackables, err := p.ByCompartmentIdProvider(compartmentId)()
	if err != nil {
		t.Fatalf("Failed to get stackables by compartment Id: %v", err)
	}

	if len(stackables) != 2 {
		t.Errorf("Expected 2 stackables for compartment, got %d", len(stackables))
	}
}

func TestWithTransaction(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	tx := db.Begin()
	defer tx.Rollback()

	pTx := p.WithTransaction(tx)
	if pTx == nil {
		t.Fatal("WithTransaction returned nil")
	}

	compartmentId := uuid.New()
	created, err := pTx.Create(compartmentId, 10, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable in transaction: %v", err)
	}

	if created.Id() == 0 {
		t.Error("Expected non-zero Id after create in transaction")
	}
}

func TestCreateWithZeroValues(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := stackable.NewProcessor(l, ctx, db)

	compartmentId := uuid.New()
	m, err := p.Create(compartmentId, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable with zero values: %v", err)
	}

	if m.Quantity() != 0 {
		t.Errorf("Expected Quantity 0, got %d", m.Quantity())
	}
	if m.OwnerId() != 0 {
		t.Errorf("Expected OwnerId 0, got %d", m.OwnerId())
	}
	if m.Flag() != 0 {
		t.Errorf("Expected Flag 0, got %d", m.Flag())
	}
	if m.Rechargeable() != 0 {
		t.Errorf("Expected Rechargeable 0, got %d", m.Rechargeable())
	}
}

func TestMultipleTenantsIsolation(t *testing.T) {
	l := testLogger()
	db := testDatabase(t)

	// Create two different tenants
	te1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	te2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)

	p1 := stackable.NewProcessor(l, ctx1, db)
	p2 := stackable.NewProcessor(l, ctx2, db)

	compartmentId := uuid.New()

	// Create stackable for tenant 1
	created1, err := p1.Create(compartmentId, 100, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable for tenant 1: %v", err)
	}

	// Create stackable for tenant 2 with same compartment ID
	created2, err := p2.Create(compartmentId, 200, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create stackable for tenant 2: %v", err)
	}

	// Verify tenant 1 can only see their stackable
	stackables1, err := p1.ByCompartmentIdProvider(compartmentId)()
	if err != nil {
		t.Fatalf("Failed to get stackables for tenant 1: %v", err)
	}
	if len(stackables1) != 1 {
		t.Errorf("Expected 1 stackable for tenant 1, got %d", len(stackables1))
	}
	if stackables1[0].Quantity() != 100 {
		t.Errorf("Expected Quantity 100 for tenant 1, got %d", stackables1[0].Quantity())
	}

	// Verify tenant 2 can only see their stackable
	stackables2, err := p2.ByCompartmentIdProvider(compartmentId)()
	if err != nil {
		t.Fatalf("Failed to get stackables for tenant 2: %v", err)
	}
	if len(stackables2) != 1 {
		t.Errorf("Expected 1 stackable for tenant 2, got %d", len(stackables2))
	}
	if stackables2[0].Quantity() != 200 {
		t.Errorf("Expected Quantity 200 for tenant 2, got %d", stackables2[0].Quantity())
	}

	// Verify tenant 1 cannot get tenant 2's stackable by ID
	_, err = p1.GetById(created2.Id())
	if err == nil {
		t.Error("Tenant 1 should not be able to get tenant 2's stackable")
	}

	// Verify tenant 2 cannot get tenant 1's stackable by ID
	_, err = p2.GetById(created1.Id())
	if err == nil {
		t.Error("Tenant 2 should not be able to get tenant 1's stackable")
	}
}
