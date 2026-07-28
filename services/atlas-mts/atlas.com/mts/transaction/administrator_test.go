package transaction_test

import (
	"atlas-mts/test"
	"atlas-mts/transaction"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func adminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, transaction.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func buildTransaction(t *testing.T, tenantId uuid.UUID, characterId uint32, kind string, createdAt time.Time) transaction.Model {
	t.Helper()
	m, err := transaction.NewBuilder(tenantId, 0, characterId).
		SetCounterpartyId(200).
		SetItemId(1302000).
		SetQuantity(3).
		SetTotalPrice(5000).
		SetKind(kind).
		SetCreatedAt(createdAt).
		Build()
	if err != nil {
		t.Fatalf("Failed to build transaction: %v", err)
	}
	return m
}

// TestAdministratorCreateGetById asserts a created transaction round-trips and
// preserves its fields.
func TestAdministratorCreateGetByCharacter(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := transaction.CreateTransaction(db, buildTransaction(t, tenantId, 100, transaction.KindPurchase, time.Now()))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("CreateTransaction did not assign an id")
	}

	got, err := transaction.NewProcessor(nil, ctx, db).GetByCharacter(100)
	if err != nil {
		t.Fatalf("GetByCharacter: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetByCharacter returned %d rows, want 1", len(got))
	}
	row := got[0]
	if row.Id() != created.Id() {
		t.Errorf("id = %s, want %s", row.Id(), created.Id())
	}
	if row.TenantId() != tenantId {
		t.Errorf("tenantId = %s, want %s", row.TenantId(), tenantId)
	}
	if row.CharacterId() != 100 {
		t.Errorf("characterId = %d, want 100", row.CharacterId())
	}
	if row.CounterpartyId() != 200 {
		t.Errorf("counterpartyId = %d, want 200", row.CounterpartyId())
	}
	if row.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", row.ItemId())
	}
	if row.Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", row.Quantity())
	}
	if row.TotalPrice() != 5000 {
		t.Errorf("totalPrice = %d, want 5000", row.TotalPrice())
	}
	if row.Kind() != transaction.KindPurchase {
		t.Errorf("kind = %q, want purchase", row.Kind())
	}
}

// TestProviderGetByCharacterOrdersNewestFirst asserts getByCharacter returns
// rows ordered by created_at DESC (newest first), the My Page -> History order.
func TestProviderGetByCharacterOrdersNewestFirst(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Insert oldest -> newest; the query must return newest -> oldest.
	oldest := buildTransaction(t, tenantId, 100, transaction.KindPurchase, base)
	middle := buildTransaction(t, tenantId, 100, transaction.KindSale, base.Add(time.Hour))
	newest := buildTransaction(t, tenantId, 100, transaction.KindPurchase, base.Add(2*time.Hour))

	for _, m := range []transaction.Model{oldest, middle, newest} {
		if _, err := transaction.CreateTransaction(db, m); err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
	}

	got, err := transaction.NewProcessor(nil, ctx, db).GetByCharacter(100)
	if err != nil {
		t.Fatalf("GetByCharacter: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("GetByCharacter returned %d rows, want 3", len(got))
	}
	if !got[0].CreatedAt().Equal(base.Add(2 * time.Hour)) {
		t.Errorf("row[0] createdAt = %s, want newest %s", got[0].CreatedAt(), base.Add(2*time.Hour))
	}
	if !got[2].CreatedAt().Equal(base) {
		t.Errorf("row[2] createdAt = %s, want oldest %s", got[2].CreatedAt(), base)
	}
}

// TestProviderGetByCharacterScopesToCharacter asserts a character only sees
// their own rows.
func TestProviderGetByCharacterScopesToCharacter(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	if _, err := transaction.CreateTransaction(db, buildTransaction(t, tenantId, 100, transaction.KindPurchase, time.Now())); err != nil {
		t.Fatalf("CreateTransaction char 100: %v", err)
	}
	if _, err := transaction.CreateTransaction(db, buildTransaction(t, tenantId, 101, transaction.KindSale, time.Now())); err != nil {
		t.Fatalf("CreateTransaction char 101: %v", err)
	}

	got, err := transaction.NewProcessor(nil, ctx, db).GetByCharacter(100)
	if err != nil {
		t.Fatalf("GetByCharacter: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetByCharacter(100) returned %d rows, want 1", len(got))
	}
}

// TestAdministratorCrossTenantIsolation asserts tenant B cannot read tenant A's
// transaction rows.
func TestAdministratorCrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := adminTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	ctxB := tenantCtx(t, tenantB)
	dbB := db.WithContext(ctxB)

	if _, err := transaction.CreateTransaction(dbA, buildTransaction(t, tenantA, 100, transaction.KindPurchase, time.Now())); err != nil {
		t.Fatalf("CreateTransaction tenant A: %v", err)
	}

	got, err := transaction.NewProcessor(nil, ctxB, dbB).GetByCharacter(100)
	if err != nil {
		t.Fatalf("tenant B GetByCharacter: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("tenant B read %d of tenant A's rows, want 0", len(got))
	}
}

// TestAdministratorIndexExists asserts the design index is created by the
// migration.
func TestAdministratorIndexExists(t *testing.T) {
	db := adminTestDB(t)
	mig := db.Migrator()

	if !mig.HasIndex(&transactionIndexProbe{}, "idx_mts_transactions_character") {
		t.Errorf("expected index %q to exist on mts_transactions", "idx_mts_transactions_character")
	}
}

type transactionIndexProbe struct{}

func (transactionIndexProbe) TableName() string { return "mts_transactions" }
