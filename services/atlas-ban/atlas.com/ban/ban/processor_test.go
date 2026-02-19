package ban

import (
	"context"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	err = db.AutoMigrate(&Entity{})
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
	return db
}

func sampleTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testContext(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func createTestBan(t *testing.T, db *gorm.DB, tm tenant.Model, banType BanType, value string, permanent bool, expiresAt time.Time) Model {
	t.Helper()
	m, err := create(db.WithContext(testContext(tm)))(tm.Id(), banType, value, "test reason", 1, permanent, expiresAt, "admin")
	if err != nil {
		t.Fatalf("Failed to create test ban: %v", err)
	}
	return m
}

func TestProcessorCreate(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	p := NewProcessor(l, ctx, db)
	m, err := p.Create(BanTypeIP, "10.0.0.1", "Cheating", 1, true, time.Time{}, "admin")
	if err != nil {
		t.Fatalf("Failed to create ban: %v", err)
	}

	if m.Type() != BanTypeIP {
		t.Errorf("BanType mismatch. Expected %v, got %v", BanTypeIP, m.Type())
	}
	if m.Value() != "10.0.0.1" {
		t.Errorf("Value mismatch. Expected 10.0.0.1, got %v", m.Value())
	}
	if m.Reason() != "Cheating" {
		t.Errorf("Reason mismatch. Expected Cheating, got %v", m.Reason())
	}
	if m.Permanent() != true {
		t.Errorf("Permanent mismatch. Expected true, got %v", m.Permanent())
	}
	if m.IssuedBy() != "admin" {
		t.Errorf("IssuedBy mismatch. Expected admin, got %v", m.IssuedBy())
	}
}

func TestProcessorGetById(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	created := createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	found, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to get ban by id: %v", err)
	}

	if found.Id() != created.Id() {
		t.Errorf("Id mismatch. Expected %v, got %v", created.Id(), found.Id())
	}
	if found.Value() != "10.0.0.1" {
		t.Errorf("Value mismatch. Expected 10.0.0.1, got %v", found.Value())
	}
}

func TestProcessorGetByIdNotFound(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	p := NewProcessor(l, ctx, db)
	_, err := p.GetById(99999)
	if err == nil {
		t.Fatal("Expected error for non-existent ban, got nil")
	}
}

func TestProcessorGetByTenant(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})
	createTestBan(t, db, st, BanTypeHWID, "HWID123", true, time.Time{})
	createTestBan(t, db, st, BanTypeAccount, "42", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	bans, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get bans by tenant: %v", err)
	}

	if len(bans) != 3 {
		t.Errorf("Expected 3 bans, got %d", len(bans))
	}
}

func TestProcessorGetByTenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st1 := sampleTenant()
	st2 := sampleTenant()

	createTestBan(t, db, st1, BanTypeIP, "10.0.0.1", true, time.Time{})
	createTestBan(t, db, st1, BanTypeIP, "10.0.0.2", true, time.Time{})
	createTestBan(t, db, st2, BanTypeIP, "10.0.0.3", true, time.Time{})

	p := NewProcessor(l, testContext(st1), db)
	bans, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get bans: %v", err)
	}

	if len(bans) != 2 {
		t.Errorf("Tenant isolation failed. Expected 2 bans for tenant 1, got %d", len(bans))
	}
}

func TestProcessorGetByType(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})
	createTestBan(t, db, st, BanTypeIP, "10.0.0.2", true, time.Time{})
	createTestBan(t, db, st, BanTypeHWID, "HWID123", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	bans, err := p.GetByType(BanTypeIP)
	if err != nil {
		t.Fatalf("Failed to get bans by type: %v", err)
	}

	if len(bans) != 2 {
		t.Errorf("Expected 2 IP bans, got %d", len(bans))
	}
}

func TestProcessorDelete(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	created := createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	err := p.Delete(created.Id())
	if err != nil {
		t.Fatalf("Failed to delete ban: %v", err)
	}

	_, err = p.GetById(created.Id())
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestProcessorCheckBanExactIP(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "192.168.1.50", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("192.168.1.50", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected ban match for exact IP, got nil")
	}
	if m.Value() != "192.168.1.50" {
		t.Errorf("Value mismatch. Expected 192.168.1.50, got %v", m.Value())
	}
}

func TestProcessorCheckBanCIDR(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.0/8", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.5.3.1", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected ban match for CIDR range, got nil")
	}
	if m.Value() != "10.0.0.0/8" {
		t.Errorf("Value mismatch. Expected 10.0.0.0/8, got %v", m.Value())
	}
}

func TestProcessorCheckBanCIDRNoMatch(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "192.168.1.0/24", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.0.0.1", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m != nil {
		t.Errorf("Expected no ban match for IP outside CIDR range, got %v", m.Value())
	}
}

func TestProcessorCheckBanHWID(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeHWID, "ABC123", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("", "ABC123", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected ban match for HWID, got nil")
	}
	if m.Value() != "ABC123" {
		t.Errorf("Value mismatch. Expected ABC123, got %v", m.Value())
	}
}

func TestProcessorCheckBanAccount(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeAccount, "42", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("", "", 42)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected ban match for account, got nil")
	}
	if m.Value() != "42" {
		t.Errorf("Value mismatch. Expected 42, got %v", m.Value())
	}
}

func TestProcessorCheckBanNoBan(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.0.0.1", "HWID123", 42)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m != nil {
		t.Errorf("Expected no ban match, got %v", m.Value())
	}
}

func TestProcessorCheckBanPriority(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})
	createTestBan(t, db, st, BanTypeHWID, "HWID123", true, time.Time{})
	createTestBan(t, db, st, BanTypeAccount, "42", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.0.0.1", "HWID123", 42)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected ban match, got nil")
	}
	if m.Type() != BanTypeIP {
		t.Errorf("Priority violation. Expected IP ban (highest priority), got type %v", m.Type())
	}
}

func TestProcessorCheckBanExpiredIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", false, time.Now().Add(-time.Hour))

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.0.0.1", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m != nil {
		t.Errorf("Expected expired ban to be ignored, got match for %v", m.Value())
	}
}

func TestProcessorCheckBanActiveTemporary(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", false, time.Now().Add(time.Hour))

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("10.0.0.1", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m == nil {
		t.Fatal("Expected active temporary ban to match, got nil")
	}
}

func TestProcessorCheckBanEmptyInputs(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	ctx := testContext(st)

	createTestBan(t, db, st, BanTypeIP, "10.0.0.1", true, time.Time{})

	p := NewProcessor(l, ctx, db)
	m, err := p.CheckBan("", "", 0)
	if err != nil {
		t.Fatalf("CheckBan failed: %v", err)
	}
	if m != nil {
		t.Errorf("Expected no match with empty inputs, got %v", m.Value())
	}
}
